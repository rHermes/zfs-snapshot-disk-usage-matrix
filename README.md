# zfs-snapshot-disk-usage-matrix

Based on https://github.com/MarcelWaldvogel/zfs-snapshot-disk-usage-matrix/tree/human-readable,
but written in go, to make certain things easier.

## Example output

The rows are `to` and the rows are `from`. Common prefixes are stripped.

```
     to\from 09-23-1120 09-29-22-25 10-01-14-26 10-02-23-00 10-05-08-49 10-07-06-01 10-23-11-06 11-03-17-32
  09-23-1120     7.4 MB                                                                                    
 09-29-22-25      24 MB       17 MB                                                                        
 10-01-14-26      25 MB       18 MB      673 kB                                                            
 10-02-23-00      26 MB       19 MB      1.6 MB      708 kB                                                
 10-05-08-49      27 MB       20 MB      2.6 MB      1.7 MB      1.0 MB                                    
 10-07-06-01      29 MB       22 MB      4.7 MB      3.9 MB      3.2 MB      1.8 MB                        
 10-23-11-06     525 GB      475 GB       42 MB       26 MB       25 MB      4.3 MB      2.4 MB            
 11-03-17-32     525 GB      475 GB       43 MB       26 MB       25 MB      4.5 MB      2.6 MB      214 kB
```
