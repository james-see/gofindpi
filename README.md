# gofindpi
Like findpi in python but in golang.

## quickstart

`go get github.com/jamesacampbell/gofindpi`

then run `sudo gofindpi`, it will ask what network you want to scan from your network device list, usually the 0 option is the correct one

It will scan each device location and store in home folder `devicesfound.txt` and `pilist.txt`

## todo

speed things up with goroutines (tried but was crashing)