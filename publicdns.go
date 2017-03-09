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
	"github.com/gocarina/gocsv"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// PublicDNSInfo is the structure that mimics the fields that belong CSV file that can be obtained from public-dns.info.
type PublicDNSInfo struct {
	// IPAddress is the ipv4 address of the server
	IPAddress string `csv:"ip"`

	// Name is the hostname of the server if the server has a hostname
	Name string `csv:"name"`

	// Country is the two-letter ISO 3166-1 alpha-2 code of the country
	Country string `csv:"country_id"`

	// City specifies the city that the server is hosted on
	City string `csv:"city"`

	// Version is the software version of the dns daemon that the server is using
	Version string `csv:"version"`

	// Error is the error that the server returned. Probably will be empty if you use the valid nameserver dataset
	Error string `csv:"error"`

	// DNSSec is a boolean to indicate if the server supports DNSSec or not
	DNSSec string `csv:"dnssec"`

	// Realiability is a normalized value - from 0.0 - 1.0 - to indicate how stable the server is
	Reliability string `csv:"reliability"`

	// CheckedAt is a timestamp to indicate the date that the server was last checked
	CheckedAt time.Time `csv:"checked_at"`

	// CreatedAt is a timestamp to indicate when the server was inserted in the database
	CreatedAt time.Time `csv:"created_at"`
}

// LoadFromFile takes a filename (assumed to be a CSV) and loads the server data contained in that file.
func LoadFromFile(filename string) ([]*PublicDNSInfo, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	servers := []*PublicDNSInfo{}
	gocsv.UnmarshalFile(file, &servers)

	return servers, nil
}

// LoadFromURL takes a URL with a CSV file, downloads the file and attempts to load the file contents using the
// previously refered LoadFromFile. A filename called nameservers.temp.csv will be created.
// TODO Delete the temporary file at the end using defer
func LoadFromURL(url string) ([]*PublicDNSInfo, error) {
	out, _ := os.Create("nameservers.temp.csv")
	defer out.Close()

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	io.Copy(out, resp.Body)

	return LoadFromFile(out.Name())
}

// DumpToDatabase dumps a complete server dataset into the selected database instance. It will create the database
// if it does not exist and insert all records present in the 'servers' variable. This function will insert all records
// in a single transaction. The test data indicates about 40000 records but the performance seems perfectly fine. Also
// consider that the table will be dropped.
//
// The database schema amounts to the same fields as the CSV value that you can find at public-dns.info.
// - IP (the ipv4 address of the server)
// - Name (the hostname of the server if the server has a hostname)
// - Country (two-letter ISO 3166-1 alpha-2 code. probably an IP location lookup by public-dns.info)
// - City (the city name that the server is hosted on. probably an IP location lookup by public-dns.info)
// - Version (the software version of the dns daemon that the server is using)
// - Error (the error that the server returned. probably will be empty if you are using the valid nameserver dataset)
// - DNSSec (boolean to indicate if the server supports DNSSec or not)
// - Reliability (a reliability value - normalized from 0.0 - 1.0 - to indicate how stable the server is)
// - CheckedAt (a timestamp to indicate the date that the server was last checked)
// - CreatedAt (a timestamp to indicate when the server was inserted in the database)
//
// TODO Create an index for Country, Reliability and IP
// TODO Fix the schema and the data types of each field to be something meaningful instead of 100% varchar
func DumpToDatabase(db *sql.DB, servers []*PublicDNSInfo) (int64, error) {
	db.Exec(`DROP TABLE nameservers`)
	db.Exec(`CREATE TABLE IF NOT EXISTS 'nameservers' (
            'ip' VARCHAR(64) PRIMARY KEY,
            'name' VARCHAR(64) NULL,
            'country' VARCHAR(64) NULL,
            'city' VARCHAR(64) NULL,
            'version' VARCHAR(64) NULL,
            'error' VARCHAR(64) NULL,
            'dnssec' VARCHAR(64) NULL,
            'reliability' VARCHAR(64) NULL,
            'checked_at' VARCHAR(64) NULL,
            'created_at' VARCHAR(64) NULL);`)

	var total int64 = 0

	tx, err := db.Begin()

	if err != nil {
		return total, err
	}

	stmt, _ := tx.Prepare("insert into nameservers(ip, name, country, city, version, error, dnssec, reliability, checked_at, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

	// TODO Should we check for an error while creating the statement or just count on the transaction to fail?
	for _, client := range servers {
		r, _ := stmt.Exec(client.IPAddress, client.Name, client.Country, client.City, client.Version, client.Error, client.DNSSec, client.Reliability, client.CheckedAt, client.CreatedAt)
		n, _ := r.RowsAffected()
		total = total + n
	}

	tx.Commit()
	return total, nil
}

// PublicDNS is the structure that is used to perform queries on the nameservers dataset that was stored in a database.
// The only parameter is an SQL connection instance. You can use any server that is supported by Golang.
type PublicDNS struct {
	db *sql.DB
}

// GetAllFromCountry obtains all the DNS servers registered in the database for a specific country. The country letter
// must be a two-letter ISO 3166-1 alpha-2 code i.e. US, PT, JP.
// TODO Do we really need to count the amount of records?
func (p *PublicDNS) GetAllFromCountry(country string) ([]*PublicDNSInfo, error) {
	count := 0
	p.db.QueryRow("SELECT COUNT(ip) FROM nameservers as n WHERE n.country = ?", country).Scan(&count)

	result, err := p.db.Query("SELECT ip, country FROM nameservers as n WHERE n.country = ?", country)

	if err != nil {
		return nil, err
	}

	defer result.Close()
	dnsinfo := make([]*PublicDNSInfo, 0)

	for result.Next() {
		info := &PublicDNSInfo{}
		result.Scan(&info.IPAddress, &info.Country)
		dnsinfo = append(dnsinfo, info)
	}

	return dnsinfo, nil

}

// GetBestFromCountry obtains the best DNS server from a specific country. This is measured by the reliability
// parameter so for many countries it will always return the same server (for the US it's always Google's DNS server).
// For countries that have less reliable DNS servers (such as those located in Africa) this could be more useful.
func (p *PublicDNS) GetBestFromCountry(country string) (*PublicDNSInfo, error) {
	result := p.db.QueryRow("select ip, country from nameservers where country = ? order by reliability DESC LIMIT 1", country)

	info := &PublicDNSInfo{}
	err := result.Scan(&info.IPAddress, &info.Country)

	if err != nil {
		return nil, err
	}

	return info, nil
}

// GetBestFromCountries takes a list of countries (two-letter ISO 3166-1 alpha-2 code) and obtains the best servers
// for each of the requested countries.
func (p *PublicDNS) GetBestFromCountries(countries []interface{}) ([]*PublicDNSInfo, error) {
	placeholders := "?" + strings.Repeat(", ?", len(countries)-1)
	stmt, err1 := p.db.Prepare("select ip, country from nameservers as n where n.country in (" + placeholders + ") group by n.country having max(n.reliability)")

	if err1 != nil {
		return nil, err1
	}

	defer stmt.Close()

	result, err2 := stmt.Query(countries...)

	if err2 != nil {
		return nil, err2
	}

	defer result.Close()

	dnsinfo := make([]*PublicDNSInfo, 0)

	for result.Next() {
		info := &PublicDNSInfo{}
		result.Scan(&info.IPAddress, &info.Country)
		dnsinfo = append(dnsinfo, info)
	}

	return dnsinfo, nil
}
