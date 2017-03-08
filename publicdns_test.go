// Package publicdns allows the user to obtain data from public-dns.info, query and manage the data
package publicdns

/*
 * The MIT License (MIT)
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
import (
    "testing"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

// TODO Make a smaller test file so we can check for the common file i/o errors and contents of a known file
func TestLoadFromFile(t *testing.T) {
    //servers, _ := LoadFromFile("nameservers.csv")
    //t.Log(len(servers))
}

// TODO Host a file somewhere to avoid using bandwidth of public-dns.info / travis-ci and also to make the test faster
func TestLoadFromURL(t *testing.T) {
    //servers, _ := LoadFromURL("http://public-dns.info/nameservers.csv")
    //t.Log(len(servers))
}

// TODO Test the actual queries and make them useful! Only GetBestFromCountries is useful for the app
func TestDumpToDatabase(t *testing.T) {
    servers, _ := LoadFromFile("nameservers.csv")
    t.Log(len(servers))

    db, _ := sql.Open("sqlite3", "./nameservers.db")
    defer db.Close()

    DumpToDatabase(db, servers)

    xx := PublicDNS{ db: db }
    xx.GetAllFromCountry("PT")

    a, _ := xx.GetBestFromCountry("PT")
    t.Log(a.IPAddress)

    b, _ := xx.GetBestFromCountries([]interface{}{"PT", "US", "CM", "JP"})
    t.Log(b)
}
