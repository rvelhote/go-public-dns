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
)

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

type PublicDNS struct {
}

func (p *PublicDNS) LoadFromFile(filename string) ([]*PublicDNSInfo, error) {
    file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)

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