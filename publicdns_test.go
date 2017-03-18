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
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"testing"
)

// TODO Make a smaller test file so we can check for the common file i/o errors and contents of a known file
func TestLoadFromFile(t *testing.T) {
	servers, err := LoadFromFile("nameservers.test.csv")

	if err != nil {
		t.Errorf("Loading the file returned the error -- %s --", err)
	}

	if len(servers) != 3 {
		t.Errorf("The test CSV file contains 3 servers. The count should have been 3 and it was -- %d --", len(servers))
	}
}

func TestLoadFailedFileLoading(t *testing.T) {
	_, err := LoadFromFile("nameservers.test.csv.does.not.exist")

	t.Log(err)
	if err == nil {
		t.Error("A non-existing file should be causing an error")
	}
}

// TODO Host a file somewhere to avoid using bandwidth of public-dns.info / travis-ci and also to make the test faster
func TestLoadFromURL(t *testing.T) {
	servers, err := LoadFromURL("https://raw.githubusercontent.com/rvelhote/go-public-dns/master/nameservers.test.csv", "nameservers.temp1.csv")

	if servers == nil || err != nil {
		t.Error("File should have been loaded from the test URL and some servers should have been processed")
		if err != nil {
			t.Log(err)
		}
	}

	// Bad URL
	_, err2 := LoadFromURL("http://does-not-exist-public-dns.info/nameservers.csv", "nameservers.temp2.csv")
	if err2 == nil {
		t.Error("Loading from a domain that does not exist should have been an error")
	}
}

// TODO Test the actual queries and make them useful! Only GetBestFromCountries is useful for the app
func TestDumpToDatabase(t *testing.T) {
	servers, _ := LoadFromFile("nameservers.test.csv")
	db, _ := sql.Open("sqlite3", "./nameservers.test.db")

	defer db.Close()

	total, err := DumpToDatabase(db, servers)

	if err != nil {
		t.Errorf("No errors should have occurred. We got -- %s --", err)
	}

	if total != 3 {
		t.Errorf("The test file contains a 3 servers so they should all have been inserted. Instead -- %d -- were inserted", total)
	}

	os.Remove("./nameservers.test.db")
}

func loadAndConnect() (*sql.DB, []*Nameserver) {
	servers, _ := LoadFromFile("nameservers.test.csv")
	db, _ := sql.Open("sqlite3", "./nameservers.test.db")
	DumpToDatabase(db, servers)
	return db, servers
}

// This test will close the database after opening it to force generate an error
func TestFailedDumpToDatabase(t *testing.T) {
	db, servers := loadAndConnect()
	db.Close()
	total, err := DumpToDatabase(db, servers)

	if total != 0 {
		t.Error("No records should have been inserted because there's no database connection")
	}

	if err == nil {
		t.Error("Error should not be nil because there's no database connection")
	}

	t.Log(err)
}

func TestPublicDNS_GetBestFromCountries(t *testing.T) {
	db, _ := loadAndConnect()
	dns := PublicDNS{DB: db}

	info, err := dns.GetBestFromCountry("DE")

	if err != nil {
		t.Errorf("GetBestFromCountry returned error -- %s --", err)
	}

	if info == nil {
		t.Error("GetBestFromCountry should have returned at least one country from Germany")
	}

	if info.IPAddress != "194.150.168.168" || info.Country != "DE" {
		t.Errorf("Should have returned -- 194.150.168.168 vs %s -- || -- DE vs %s --", info.IPAddress, info.Country)
	}
}

func TestPublicDNS_GetAllFromCountry(t *testing.T) {
	db, _ := loadAndConnect()
	dns := PublicDNS{DB: db}

	info, err := dns.GetAllFromCountry("US")

	if err != nil {
		t.Errorf("GetAllFromCountry returned error -- %s --", err)
	}

	if info == nil {
		t.Error("GetAllFromCountry should not have returned nil because there are countries in the database")
	}

	if len(info) != 2 {
		t.Errorf("Should have returned 2 servers but returned -- %d --", len(info))
	}
}

func TestPublicDNS_GetBestFromCountry(t *testing.T) {
	db, _ := loadAndConnect()
	dns := PublicDNS{DB: db}

	info, err := dns.GetBestFromCountries([]interface{}{"US", "DE"})

	if err != nil {
		t.Errorf("GetBestFromCountries returned error -- %s --", err)
	}

	if len(info) != 2 {
		t.Errorf("GetBestFromCountries should have returned 2 servers but returned -- %d --", len(info))
	}

	if len(info) == 2 && (info[0].Country == info[1].Country) {
		t.Errorf("GetBestFromCountries should have returned two different countries -- %s vs %s --", info[0].Country, info[1].Country)
	}

}
