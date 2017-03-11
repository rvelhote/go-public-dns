[![](https://godoc.org/github.com/rvelhote/go-public-dns?status.svg)](https://godoc.org/github.com/rvelhote/go-public-dns) [![Build Status](https://travis-ci.org/rvelhote/go-public-dns.svg?branch=master)](https://travis-ci.org/rvelhote/go-public-dns) [![codecov](https://codecov.io/gh/rvelhote/go-public-dns/branch/master/graph/badge.svg)](https://codecov.io/gh/rvelhote/go-public-dns) [![Code Climate](https://lima.codeclimate.com/github/rvelhote/go-public-dns/badges/gpa.svg)](https://lima.codeclimate.com/github/rvelhote/go-public-dns) [![Issue Count](https://codeclimate.com/github/rvelhote/go-public-dns/badges/issue_count.svg)](https://codeclimate.com/github/rvelhote/go-public-dns)

# Public DNS Info
This package allows you to interact with the information about Nameservers worldwide contained in the website [public-dns-info](http://public-dns.info). It allows you to download the content of the CSV files that public-dns.info makes available and dump its content into a queryable database (in the tests SQLite is used).

Its primary use case is to allow automatic server selection as well as health checking to my other project where you can [check the propagation](https://github.com/rvelhote/dnspropagation) of DNS records worldwide. Some nameservers are in locations with unstable connections so this library is a way to always obtain the most reliable server. In Africa, for example, it's common that servers go offline or new, more reliable ones come up.

Due to the fact that the library has a very specific use case its features are merely the minimum required for the purposes of the DNS propagation project.

## Usage
The main workflow to use this library is the following:
1. Get the file with the list of nameservers from a URL or use a cached one
2. Parse the contents of the loaded file (done during the fetch phase)
3. Dump the contents into a database of your choice. The database is always destroyed in every dump
3. Query the database using the helper method from the package

```
import "github.com/rvelhote/go-public-dns"

servers, _ := publicdns.LoadFromFile("nameservers.test.csv")
servers2, _ := publicdns.LoadFromFile("nameservers.test.csv.does.not.exist")

// Open a database connection. Feel free to use whatever driver you desire
// NOTE: Only tested in SQLite so far
db, _ := sql.Open("sqlite3", "nameservers.test.db")
defer db.Close()

total, _ := publicdns.DumpToDatabase(db, servers)

dnsquery := publicdns.PublicDNS{db: db}
info, _ := dnsquery.GetBestFromCountry("DE")

fmt.Sprintf("IP: %s Country: %s", info.IPAddress, info.Country)
```

## Further Developments
- Test other SQL drivers to make sure we have freedom in using whatever database engine
- Add more functions with useful queries

## Contributing
You are very welcome to contribute with issues/requests and pull requests if you find this library useful.