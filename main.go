/*
Copyright 2021 Teodor Spæren

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/dustin/go-humanize"
)

var (
	rawFlag     = flag.Bool("raw", false, "If set, get numbers exactly.")
	hostFlag    = flag.String("host", "", "If set, the hostname which we should ssh into to do the commands")
	nameFlag    = flag.String("name", "", "The name of the dataset to scan")
	cmdFlag     = flag.String("cmd", "/sbin/zfs", "The path to the zfs cmd")
	recurseFlag = flag.Bool("recurse", false, "Calculate if snapshots where recursively deleted")
)

type Pair struct {
	From string
	To   string
}

// A Dataset
type Dataset struct {
	host string
	name string
}

func (d *Dataset) zfsDefs() (string, []string) {
	if d.host == "" {
		return *cmdFlag, []string{}
	}
	return "ssh", []string{d.host, *cmdFlag}
}

func (d *Dataset) snapshotByCreation() ([]string, error) {
	cmd, args := d.zfsDefs()
	specArgs := []string{
		"list",
		"-H",
		"-t", "snapshot",
		"-o", "name",
		"-s", "creation",
		d.name,
	}
	finalArgs := append(args, specArgs...)

	c := exec.Command(cmd, finalArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stderr = &stderr
	c.Stdout = &stdout

	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("executing command: %v: %s", err, stderr.String())
	}

	snaps := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	// strip the names from the snapshot
	for i, name := range snaps {
		snaps[i] = strings.TrimPrefix(name, d.name+"@")
	}

	return snaps, nil
}

func (d *Dataset) spaceBetweenSnapshots(from, to string) (uint64, error) {
	cmd, args := d.zfsDefs()
	specArgs := []string{
		"destroy", "-nvp",
	}
	if *recurseFlag {
		specArgs = append(specArgs, "-r")
	}

	specArgs = append(specArgs, fmt.Sprintf("%s@%s%%%s", d.name, from, to))
	finalArgs := append(args, specArgs...)

	c := exec.Command(cmd, finalArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stderr = &stderr
	c.Stdout = &stdout

	// t0 := time.Now()
	if err := c.Run(); err != nil {
		return 0, fmt.Errorf("executing command: %v: %s", err, stderr.String())
	}

	re := regexp.MustCompile(`(?m)^reclaim\t(0|[1-9][0-9]*)$`)
	mtch := re.FindStringSubmatch(stdout.String())
	if len(mtch) != 2 {
		return 0, fmt.Errorf("couldn't match line")
	}

	ans, err := strconv.ParseUint(mtch[1], 10, 64)
	if err != nil {
		panic("this should never happen")
	}

	// dur := time.Since(t0)
	// log.Printf("[%s] -> [%s] took %s and returned %d", from, to, dur.String(), ans)
	return ans, nil
}

func (d *Dataset) getAllCombs(snaps []string) (map[Pair]uint64, error) {
	ans := make(map[Pair]uint64)

	for i, to := range snaps {
		for _, from := range snaps[:i+1] {
			spd, err := d.spaceBetweenSnapshots(from, to)
			if err != nil {
				return nil, err
			}
			ans[Pair{From: from, To: to}] = spd
		}
	}

	return ans, nil
}

func (d *Dataset) getAllCombsEx(snaps []string) (map[Pair]uint64, error) {
	// We only use this for external hosts.
	if d.host == "" {
		return d.getAllCombs(snaps)
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd := exec.Command("ssh", d.host)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	order := []Pair{}
	shortFlags := "pn"
	if *recurseFlag {
		shortFlags += "r"
	}
	for i, to := range snaps {
		for _, from := range snaps[:i+1] {
			fmt.Fprintf(stdin, "/sbin/zfs destroy -%s %s@%s%%%s\n", shortFlags, d.name, from, to)
			order = append(order, Pair{From: from, To: to})
		}
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(?m)^reclaim\t(0|[1-9][0-9]*)$`)
	matches := re.FindAllStringSubmatch(stdout.String(), -1)
	if len(matches) != len(order) {
		return nil, fmt.Errorf("not enough matches: %s", stderr.String())
	}

	ret := make(map[Pair]uint64)
	for i, match := range matches {
		ans, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			panic("this should never happen")
		}

		ret[order[i]] = ans
	}

	return ret, nil
}

func (d *Dataset) SavingMatrix() (string, error) {
	snaps, err := d.snapshotByCreation()
	if err != nil {
		return "", err
	}

	pairs, err := d.getAllCombsEx(snaps)
	if err != nil {
		return "", err
	}

	// Remove common prefixes, to make it easier to read the output
	fsnaps := TrimPrefix(snaps)

	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.AlignRight)

	fmt.Fprintf(w, "to\\from")

	// Write the header
	for _, snap := range fsnaps {
		fmt.Fprintf(w, "\t%s", snap)
	}
	fmt.Fprintf(w, "\t\n")

	// We calculate the savings
	for i, to := range snaps {
		fmt.Fprintf(w, "%s", fsnaps[i])
		for j, from := range snaps {
			if i < j {
				fmt.Fprintf(w, "\t")
				continue
			}

			spd, ok := pairs[Pair{From: from, To: to}]
			if !ok {
				panic("this is not supposed to happen")
			}

			if *rawFlag {
				fmt.Fprintf(w, "\t%d", spd)
			} else {
				fmt.Fprintf(w, "\t%s", humanize.Bytes(spd))
			}

		}
		fmt.Fprintln(w, "\t")
	}
	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("tabwriter flush: %v", err)
	}

	return buf.String(), nil
}

func main() {
	flag.Parse()

	if *nameFlag == "" {
		flag.Usage()
		return
	}

	d := Dataset{
		host: *hostFlag,
		name: *nameFlag,
	}

	mat, err := d.SavingMatrix()
	if err != nil {
		log.Fatalf("saving matrix: %v", err)
	}

	fmt.Print(mat)
}
