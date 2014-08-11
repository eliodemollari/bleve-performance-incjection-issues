//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/couchbaselabs/bleve"
	bleveHttp "github.com/couchbaselabs/bleve/http"
)

var bindAddr = flag.String("addr", ":8094", "http listen address")
var jsonDir = flag.String("jsonDir", "../../samples/beer-sample/", "json directory")
var indexDir = flag.String("indexDir", "beer-search.bleve", "index directory")
var staticEtag = flag.String("staticEtag", "", "A static etag value.")
var staticPath = flag.String("static", "static/", "Path to the static content")

func main() {

	flag.Parse()

	// create a mapping
	indexMapping := buildIndexMapping()

	// open the index
	beerIndex, err := bleve.Open(*indexDir, indexMapping)
	if err != nil {
		log.Fatal(err)
	}

	// index data in the background
	go func() {
		err = indexBeer(beerIndex)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// create a router to serve static files
	router := staticFileRouter()

	// add the API
	bleveHttp.RegisterIndexName("beer", beerIndex)
	searchHandler := bleveHttp.NewSearchHandler("beer")
	router.Handle("/api/search", searchHandler).Methods("POST")
	listFieldsHandler := bleveHttp.NewListFieldsHandler("beer")
	router.Handle("/api/fields", listFieldsHandler).Methods("GET")
	debugHandler := bleveHttp.NewDebugDocumentHandler("beer")
	router.Handle("/api/debug/{docID}", debugHandler).Methods("GET")

	// start the HTTP server
	http.Handle("/", router)
	log.Printf("Listening on %v", *bindAddr)
	log.Fatal(http.ListenAndServe(*bindAddr, nil))

}

func indexBeer(i bleve.Index) error {

	// open the directory
	dirEntries, err := ioutil.ReadDir(*jsonDir)
	if err != nil {
		return err
	}

	// walk the directory entries for indexing
	log.Printf("Indexing...")
	count := 0
	startTime := time.Now()
	for _, dirEntry := range dirEntries {
		filename := dirEntry.Name()
		// read the bytes
		jsonBytes, err := ioutil.ReadFile(*jsonDir + "/" + filename)
		if err != nil {
			return err
		}
		// // shred them into a document
		ext := filepath.Ext(filename)
		docId := filename[:(len(filename) - len(ext))]
		err = i.Index(docId, jsonBytes)
		if err != nil {
			return err
		}
		count++
		if count%1000 == 0 {
			indexDuration := time.Since(startTime)
			indexDurationSeconds := float64(indexDuration) / float64(time.Second)
			timePerDoc := float64(indexDuration) / float64(count)
			log.Printf("Indexed %d documents, in %.2fs (average %.2fms/doc)", count, indexDurationSeconds, timePerDoc/float64(time.Millisecond))
		}
	}
	indexDuration := time.Since(startTime)
	indexDurationSeconds := float64(indexDuration) / float64(time.Second)
	timePerDoc := float64(indexDuration) / float64(count)
	log.Printf("Indexed %d documents, in %.2fs (average %.2fms/doc)", count, indexDurationSeconds, timePerDoc/float64(time.Millisecond))
	return nil
}
