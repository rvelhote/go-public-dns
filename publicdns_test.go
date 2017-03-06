// Package publicdns allows the user to obtain data from public-dns.info, query and manage the data
package publicdns

import (
    "testing"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

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
func TestPublicDNS_LoadFromFile(t *testing.T) {
    dnsinfo := PublicDNS{}
    servers, _ := dnsinfo.LoadFromFile("nameservers.csv")

    t.Log(len(servers))



    db, _ := sql.Open("sqlite3", "./nameservers.db")
    defer db.Close()

    db.Exec("CREATE TABLE `nameservers` (`ip` VARCHAR(64) PRIMARY KEY, `name` VARCHAR(64) NULL,`country` VARCHAR(64) NULL,`city` VARCHAR(64) NULL, `version` VARCHAR(64) NULL, `error` VARCHAR(64) NULL, `dnssec` VARCHAR(64) NULL, `reliability` VARCHAR(64) NULL, `checked_at` VARCHAR(64) NULL, `created_at` VARCHAR(64) NULL);")



    tx, _ := db.Begin()
    stmt, _ := tx.Prepare("insert into nameservers(ip, name, country, city, version, error, dnssec, reliability, checked_at, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")


    for _, client := range servers {
        stmt.Exec(client.IPAddress, client.Name, client.Country, client.City, client.Version, client.Error, client.DNSSec, client.Reliability, client.CheckedAt, client.CreatedAt)

    }

    tx.Commit()
}