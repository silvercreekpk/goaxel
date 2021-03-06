/*                                                                              
 * Copyright (C) 2013 Deepin, Inc.                                                 
 *               2013 Leslie Zhai <zhaixiang@linuxdeepin.com>                   
 *                                                                              
 * This program is free software: you can redistribute it and/or modify         
 * it under the terms of the GNU General Public License as published by         
 * the Free Software Foundation, either version 3 of the License, or            
 * any later version.                                                           
 *                                                                              
 * This program is distributed in the hope that it will be useful,              
 * but WITHOUT ANY WARRANTY; without even the implied warranty of               
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                
 * GNU General Public License for more details.                                 
 *                                                                              
 * You should have received a copy of the GNU General Public License            
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.        
 */

package main

import (
    "fmt"
    "flag"
    "os"
    "net/url"
    "strings"
    "strconv"
    "path"
    "sync"
    "github.com/xiangzhai/goaxel/conn"
)

const (
    appName                 string = "GoAxel"
    defaultOutputFileName   string = "default"
)

var (
    connNum         uint
    userAgent       string
    debug           bool
    urls            []string
    outputFileName  string
    outputFile      *os.File
    protocol        string
    host            string
    port            int
    userName        string  = ""
    passwd          string  = ""
    strPath         string
    contentLength   int
    acceptRange     bool
    received        int     = 0
    chunkFileName   []string
    wg              sync.WaitGroup
)

func init() {
    flag.UintVar(&connNum, "n", 3, "Specify maximum speed (bytes per second)")
    flag.StringVar(&outputFileName, "o", defaultOutputFileName, "Specify local output file")
    flag.StringVar(&userAgent, "U", appName, "Set user agent")
    flag.BoolVar(&debug, "d", false, "Debug")
}

func connCallback(n int) {
    received += n
    fmt.Println("received:", received)
}

func startRoutine(range_from, range_to int) {
    defer wg.Done()
    conn := &conn.CONN{Protocol: protocol, Host: host, Port: port, UserAgent: userAgent, UserName: userName, Passwd: passwd, Path: strPath, Debug: debug, Callback: connCallback}
    conn.Get(range_from, range_to, outputFile, outputFileName)
}

/* TODO: parse url to get host, port, path, basename */
func parseUrl(strUrl string) {
    u, err := url.Parse(strUrl)
    if err != nil {
        fmt.Println("ERROR:", err.Error())
        return
    }
    protocol = u.Scheme
    host = u.Host
    port = 80
    if protocol == "https" {
        port = 443
    } else if protocol == "ftp" {
        port = 21
    }
    userinfo := u.User
    if userinfo != nil {
        userName = userinfo.Username()
        passwd, _ = userinfo.Password()
    }
    strPath = u.Path
    pos := strings.Index(host, ":")
    if pos != -1 {
        port, _ = strconv.Atoi(host[pos + 1:])
        host = host[0:pos]
    }
    conn := &conn.CONN{Protocol: protocol, Host: host, Port: port, UserAgent: userAgent, UserName: userName, Passwd: passwd, Path: strPath, Debug: debug}
    if outputFileName == defaultOutputFileName && path.Base(strPath) != "/" {
        outputFileName = path.Base(strPath)
    }
    contentLength, acceptRange = conn.GetContentLength(outputFileName)
    if debug {
        fmt.Println("DEBUG: output filename", outputFileName)
        fmt.Println("DEBUG: content length", contentLength)
    }
}

func splitWork() {
    offset := contentLength / int(connNum)
    remainder := contentLength % (offset * int(connNum))
    start := 0
    for i := 0; i < int(connNum); i++ {
        go startRoutine(start, start + offset - 1)
        start += offset
        if (i == int(connNum) - 2) {
            offset += remainder
        }
    }
}

func main() {
    if len(os.Args) == 1 {
        fmt.Println("Usage: goaxel [options] url1 [url2] [url...]")
        return
    }

    if os.Args[1] == "-V" {
        fmt.Println(fmt.Sprintf("%s Version 1.0", appName))
        fmt.Println("Copyright 2013 Leslie Zhai")
        return
    }

    flag.Parse()

    for i := 0; i < len(flag.Args()); i++ {
        urls = append(urls, flag.Args()[i])
    }
    if len(urls) == 0 {
        fmt.Println("Invalid urls")
        return
    }
    if len(urls) == 1 {
        parseUrl(urls[0])
    }

    outputFile, _ = os.Create(outputFileName)
    defer outputFile.Close()

    if acceptRange && connNum != 1 {
        wg.Add(int(connNum))
        splitWork()
    } else {
        wg.Add(1)
        fmt.Println("It does not accept range, use signal connection instead")
        go startRoutine(0, 0)
    }
    wg.Wait()
}
