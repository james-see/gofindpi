# gofindpi
Like findpi in python but in golang.

## quickstart

`go get github.com/james-see/gofindpi`

then simply run `gofindpi`, it will ask what network you want to scan from your network device list, usually the 0 option is the correct one

_Note: you will be asked for your password during the awk portion of the command_

It will scan each device location and store in home folder `devicesfound.txt` and `pilist.txt`

## benchmarks

Note: To compare apples to apples, I wiped the local arp cache for each trial by running `sudo arp -d -a`.

Ok, so to compare this to just running nmap vs. [findpi](https://github.com/jamesacampbell/findpi) vs gofindpi:

|               | run 1       | run 2       | run 3       | average    |
|---------------|-------------|-------------|-------------|------------|
| nmap v7.80    | 6.007 total | 5.679 total | 4.633 total | 5.44 total |
| findpi v1.0.3 | 2.899 total | 2.682 total | 2.696 total | 2.76 total |
| *gofindpi v1.0.3* | *0.987 total* | *0.943 total* | *0.981 total* | *0.97 total* |

Note: these are conservative benchmarks, the latest version is exhibiting 0.09 seconds repeatedly and I want to make sure I am not doing something wrong before I publish those numbers.
