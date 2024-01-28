# Gobuster
Gobuster is a scanning tool written in Golang.

## Installation

Windows

Download and install last go version : https://go.dev/dl/go1.21.6.windows-amd64.msi

Mac OS

```bash
brew install go
```
Linux

```bash
wget https://dl.google.com/go/go1.13.5.linux-amd64.tar.gz
	
sudo tar -C /usr/local/ -xzf go1.13.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

```

## Usage

```bash
go run gobuster.go -h : Show help
go build gobuster.go
```

Options available :

```bash
-l  Log mode : Log results to a file
-q  Quiet mode : Only show HTTP 200 (Default)

-d  Path to dictionary file (Mandatory)
-t  Target to enumerate (Mandatory)
-w  Number of workers to run (Default 1)

```

## Examples

```bash
go run gobuster.go -d wordlist.txt -t https://randomsite.com -w 120
Checking connectivity (HTTPS)... Failed
Checking connectivity (HTTP)... OK
---
Target: http://randomsite.com
List: wordlist.txt
Dictionary Size: 3521
Workers: 120
---
Starting scan...
http://randomsite.com/robots.txt 200
Scan done in 18.073891s
```

## Author
Aymen H.

## License

[Apache](http://www.apache.org/licenses/)