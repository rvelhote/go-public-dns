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
	"errors"
	"fmt"
	"github.com/gocarina/gocsv"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// NameserverCountryTally is a structure that will hold the result of the GetTotalServersPerCountry func
type NameserverCountryTally struct {
	Country string
	Total int
}

// Nameserver is the structure that mimics the fields that belong CSV file that can be obtained from public-dns.info.
type Nameserver struct {
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
	DNSSec bool `csv:"dnssec"`

	// Realiability is a normalized value - from 0.0 - 1.0 - to indicate how stable the server is
	Reliability string `csv:"reliability"`

	// CheckedAt is a timestamp to indicate the date that the server was last checked
	CheckedAt time.Time `csv:"checked_at"`

	// CreatedAt is a timestamp to indicate when the server was inserted in the database
	CreatedAt time.Time `csv:"created_at"`
}

// LoadFromFile takes a filename (assumed to be a CSV) and loads the server data contained in that file.
func LoadFromFile(filename string) ([]*Nameserver, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	servers := []*Nameserver{}
	err = gocsv.UnmarshalFile(file, &servers)

	if err != nil {
		return nil, err
	}

	return servers, nil
}

// LoadFromURL takes a URL with a CSV file, downloads the file and attempts to load the file contents using the
// previously refered LoadFromFile. A filename called nameservers.temp.csv will be created.
func LoadFromURL(url string, filename string) ([]*Nameserver, error) {
	out, err := os.Create(filename)

	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	written, err := io.Copy(out, resp.Body)

	if err != nil {
		return nil, err
	}

	if written == 0 {
		return nil, errors.New("No bytes written")
	}

	err = out.Sync()
	if err != nil {
		return nil, err
	}

	out.Close()

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
func DumpToDatabase(db *sql.DB, servers []*Nameserver) (int64, error) {
	var total int64
	var query string
	var fields []string

	// It's safe to ignore the execution error that my occur.
	// If there is an problem with the deletion it will be thrown in the table creation query
	db.Exec(`DROP TABLE 'nameservers'`)

	fields = []string{
		`'ip' VARCHAR(45) PRIMARY KEY`,
		`'name' VARCHAR(64) NULL`,
		`'country' VARCHAR(2) NULL`,
		`'city' VARCHAR(64) NULL`,
		`'version' VARCHAR(16) NULL`,
		`'error' VARCHAR(256) NULL`,
		`'dnssec' TINYINT NULL`,
		`'reliability' FLOAT NULL`,
		`'checked_at' DATETIME NULL`,
		`'created_at' DATETIME NULL`,
	}

	query = `CREATE TABLE IF NOT EXISTS 'nameservers' (` + strings.Join(fields, ",") + `);`
	_, errCreateTable := db.Exec(query)

	if errCreateTable != nil {
		return total, errCreateTable
	}

	indexes := []string{
		"CREATE INDEX nameservers_country_index ON nameservers(country);",
		"CREATE INDEX nameservers_country_reliability_index ON nameservers(country,reliability);",
		"CREATE INDEX nameservers_reliability_index ON nameservers(reliability);",
	}

	_, errCreateIndexes := db.Exec(strings.Join(indexes, ""))

	if errCreateIndexes != nil {
		return total, errCreateIndexes
	}

	tx, err := db.Begin()

	if err != nil {
		return total, err
	}

	fields = []string{
		"ip",
		"name",
		"country",
		"city",
		"version",
		"error",
		"dnssec",
		"reliability",
		"checked_at",
		"created_at",
	}

	query = "INSERT INTO nameservers(" + strings.Join(fields, ",") + ") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	stmt, prepareErr := tx.Prepare(query)

	if prepareErr != nil {
		return total, prepareErr
	}

	// TODO Should we check for an error while creating the statement or just count on the transaction to fail?
	for _, client := range servers {
		r, _ := stmt.Exec(
			client.IPAddress,
			client.Name,
			client.Country,
			client.City,
			client.Version,
			client.Error,
			client.DNSSec,
			client.Reliability,
			client.CheckedAt,
			client.CreatedAt,
		)

		n, _ := r.RowsAffected()
		total = total + n
	}

	if txErr := tx.Commit(); txErr != nil {
		tx.Rollback()
		return 0, txErr
	}

	return total, nil
}

// PublicDNS is the structure that is used to perform queries on the nameservers dataset that was stored in a database.
// The only parameter is an SQL connection instance. You can use any server that is supported by Golang.
type PublicDNS struct {
	DB *sql.DB
}

// GetAllFromCountry obtains all the DNS servers registered in the database for a specific country. The country letter
// must be a two-letter ISO 3166-1 alpha-2 code i.e. US, PT, JP.
// TODO Do we really need to count the amount of records?
func (p *PublicDNS) GetAllFromCountry(country string) ([]*Nameserver, error) {
	count := 0
	p.DB.QueryRow("SELECT COUNT(ip) FROM nameservers as n WHERE n.country = ?", country).Scan(&count)

	result, err := p.DB.Query("SELECT ip, country, city FROM nameservers as n WHERE n.country = ?", country)

	if err != nil {
		return nil, err
	}

	defer result.Close()

	var dnsinfo []*Nameserver

	for result.Next() {
		info := &Nameserver{}
		result.Scan(&info.IPAddress, &info.Country, &info.City)
		dnsinfo = append(dnsinfo, info)
	}

	return dnsinfo, nil

}

// GetBestFromCountry obtains the best DNS server from a specific country. This is measured by the reliability
// parameter so for many countries it will always return the same server (for the US it's always Google's DNS server).
// For countries that have less reliable DNS servers (such as those located in Africa) this could be more useful.
func (p *PublicDNS) GetBestFromCountry(country string) (*Nameserver, error) {
	result := p.DB.QueryRow("SELECT ip, country, city FROM nameservers WHERE country = ? ORDER BY reliability DESC LIMIT 1", country)

	info := &Nameserver{}
	err := result.Scan(&info.IPAddress, &info.Country, &info.City)

	if err != nil {
		return nil, err
	}

	return info, nil
}

// GetBestFromCountries takes a list of countries (two-letter ISO 3166-1 alpha-2 code) and obtains the best servers
// for each of the requested countries.
func (p *PublicDNS) GetBestFromCountries(countries []interface{}) ([]*Nameserver, error) {
	// This will create someting like IN(?, ?, ?) (depending on the number of countries)
	placeholders := "?" + strings.Repeat(", ?", len(countries)-1)

	subquery := "SELECT n.ip, n.country, n.city " +
		"FROM nameservers AS n " +
		"WHERE n.country IN (" + placeholders + ")  and name != '' and city != '' AND reliability = 1 " +
		"ORDER BY reliability ASC, n.checked_at ASC"
	query := fmt.Sprintf("SELECT * FROM (%s) as nn GROUP BY nn.country;", subquery)

	stmt, err1 := p.DB.Prepare(query)

	if err1 != nil {
		return nil, err1
	}

	defer stmt.Close()

	// Then, using the variadic operator, we expand the list of countries into the placeholders
	result, err2 := stmt.Query(countries...)

	if err2 != nil {
		return nil, err2
	}

	defer result.Close()

	var dnsinfo []*Nameserver

	for result.Next() {
		info := &Nameserver{}
		result.Scan(&info.IPAddress, &info.Country, &info.City)
		dnsinfo = append(dnsinfo, info)
	}

	return dnsinfo, nil
}

// GetNameserverPerCountryTally obtains a list the total of "good" nameservers that exist per country. In this context
// "good" means that the server as a hostname (reverse lookup) has city name and its reliability score is 1 (maximum).
func (p *PublicDNS) GetNameserverPerCountryTally() ([]*NameserverCountryTally, error) {
	query := "SELECT n.country AS Country, COUNT(n.ip) AS Total " +
		"FROM nameservers AS n " +
		"WHERE n.name != '' AND n.city != '' AND n.reliability = 1 " +
		"GROUP BY n.country"

	rows, err := p.DB.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var nameserverTally []*NameserverCountryTally

	for rows.Next() {
		tally := &NameserverCountryTally{}
		rows.Scan(&tally.Country, &tally.Total)
		nameserverTally = append(nameserverTally, tally)
	}

	return nameserverTally, nil
}
