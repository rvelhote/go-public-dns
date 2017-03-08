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
    "time"
    "os"
    "github.com/gocarina/gocsv"
    "net/http"
    "io"
    "database/sql"
    "strings"
)

const TEMP_FILENAME = "nameservers.temp.csv"
const CREATE_QUERY = `
CREATE TABLE IF NOT EXISTS 'nameservers' (
    'ip' VARCHAR(64) PRIMARY KEY,
    'name' VARCHAR(64) NULL,
    'country' VARCHAR(64) NULL,
    'city' VARCHAR(64) NULL,
    'version' VARCHAR(64) NULL,
    'error' VARCHAR(64) NULL,
    'dnssec' VARCHAR(64) NULL,
    'reliability' VARCHAR(64) NULL,
    'checked_at' VARCHAR(64) NULL,
    'created_at' VARCHAR(64) NULL
);
`

type PublicDNSInfo struct {
    IPAddress string `csv:"ip"`
    Name string `csv:"name"`
    Country string `csv:"country_id"`
    City string `csv:"city"`
    Version string `csv:"version"`
    Error string `csv:"error"`
    DNSSec string `csv:"dnssec"`
    Reliability string `csv:"reliability"`
    CheckedAt time.Time `csv:"checked_at"`
    CreatedAt time.Time `csv:"created_at"`
}

func LoadFromFile(filename string) ([]*PublicDNSInfo, error) {
    file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)

    if err != nil {
        return nil, err
    }

    defer file.Close()

    servers := []*PublicDNSInfo{}
    err = gocsv.UnmarshalFile(file, &servers)

    if err != nil {
        return nil, err
    }

    return servers, nil
}

func LoadFromURL(url string) ([]*PublicDNSInfo, error) {
    out, err := os.Create(TEMP_FILENAME)
    if err != nil {
        return nil, err
    }

    defer out.Close()

    resp, err := http.Get(url)

    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return nil, err
    }

    return LoadFromFile(out.Name())
}

func DumpToDatabase(db *sql.DB, servers []*PublicDNSInfo) error {
    db.Exec(CREATE_QUERY)

    tx, _ := db.Begin()
    stmt, _ := tx.Prepare("insert into nameservers(ip, name, country, city, version, error, dnssec, reliability, checked_at, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

    for _, client := range servers {
        stmt.Exec(client.IPAddress, client.Name, client.Country, client.City, client.Version, client.Error, client.DNSSec, client.Reliability, client.CheckedAt, client.CreatedAt)
    }

    tx.Commit()
    return nil
}


type PublicDNS struct {
    db *sql.DB
}

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

func (p *PublicDNS) GetBestFromCountry(country string) (*PublicDNSInfo, error) {
    result := p.db.QueryRow("select ip, country from nameservers where country = ? order by reliability DESC LIMIT 1", country)

    info := &PublicDNSInfo{}
    err := result.Scan(&info.IPAddress, &info.Country)

    if err != nil {
        return nil, err
    }

    return info, nil
}

func (p *PublicDNS) GetBestFromCountries(countries []interface{}) ([]*PublicDNSInfo, error) {
    placeholders := "?" + strings.Repeat(", ?", len(countries) - 1)
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
