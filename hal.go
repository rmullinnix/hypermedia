//Copyright 2014  (rmullinnix@yahoo.com). All rights reserved.
//
//Redistribution and use in source and binary forms, with or without
//modification, are permitted provided that the following conditions
//are met:
//
//  1. Redistributions of source code must retain the above copyright
//     notice, this list of conditions and the following disclaimer.
//
//  2. Redistributions in binary form must reproduce the above copyright
//     notice, this list of conditions and the following disclaimer
//     in the documentation and/or other materials provided with the
//     distribution.
//
//THIS SOFTWARE IS PROVIDED BY THE AUTHOR ``AS IS'' AND ANY EXPRESS OR
//IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES
//OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
//IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
//SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
//PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS;
//OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
//WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR
//OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF
//ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


package hypermedia

import (
	"strings"
	"reflect"
	"unicode"
)

type HalDocument 	map[string]interface{}

type HalCurie struct {
	Name		string			`json:"name"`
	Href		string			`json:"href"`
	Templated	bool			`json:"templated,omitempty"`
}

type HalLink struct {
	Href		string			`json:"href"`
	Templated	bool			`json:"templated,omitempty"`
	Type		string			`json:"type,omitempty"`
	Deprecation	string			`json:"deprecation,omitempty"`
	Name		string			`json:"name,omitempty"`
	Profile		string			`json:"profile,omitempty"`
	Title		string			`json:"title,omitempty"`
	Hreflang	string			`json:"hreflang,omitempty"`
}

// This takes the data destined for the http response body and adds hypermedia content
// to the message prior to marshaling the data and returning it to the client
// The HalDecorator loosely follows the HAL specification
// mime type: applcation/hal+json 
func halDecorator(prefix string, response interface{}) (interface{}) {
	var hm_resp 	HalDocument

	hm_resp = make(map[string]interface{}, 0)
	links := make(map[string]interface{}, 1)
	v := reflect.ValueOf(response)
	switch v.Kind() {
		case reflect.Struct:
			// Properties - not sub-entity items
			// Any sub-entities (struct or array), placed in Embedded
			class := reflect.TypeOf(response).Name()
			key := "" //v.FieldByName(entities[class].key)

			halResourceLinks(links, entities[class], key, false)
			halDocumentCuries(links, entities[class])

			hm_resp["_links"] = links

			stripEmbedded(hm_resp, v, entities)
			//hm_resp = append(hm_resp, properties)
			//hm_resp = append(hm_resp, embedded)
		case reflect.Slice, reflect.Array, reflect.Map:
			//hm_resp.Embedded.Resources, _ = getEmbeddedList(v, entities)
		default:
			//hm_resp.Properties = response
			//hm_resp.Class = reflect.TypeOf(response).Name()
	}

	return hm_resp
}

func halResourceLinks(lnklist map[string]interface{}, ent entity, keyval string, sub bool) {
	
	for _, e_lnk := range ent.links {
		if sub && strings.IndexFunc(e_lnk.rel[:1], unicode.IsUpper) == 0 {
			continue
		}

		lnk := HalLink{e_lnk.href, e_lnk.templated, e_lnk.typ, "", e_lnk.name, "", e_lnk.title, ""}
		if strings.Contains(lnk.Href, "{key}") {
			lnk.Href = strings.Replace(lnk.Href, "{key}", keyval, 1)
		}
		lnklist[e_lnk.rel] = lnk
	}
}

func halDocumentCuries(lnklist map[string]interface{}, ent entity) {
	var curlist	[]HalCurie

	curlist = make([]HalCurie, len(ent.curies))
	i := 0
	for _, e_cur := range ent.curies {
		cur := HalCurie{e_cur.name, e_cur.href, e_cur.templated}
		curlist[i] = cur
		i++
	}
	if i > 0 {
		lnklist["curies"] = curlist
	}
}

func stripEmbedded(resp map[string]interface{}, in reflect.Value, entities map[string]entity) {
	emb := make(map[string]interface{}, 0)

	typ := reflect.TypeOf(in.Interface())
	for i := 0; i < typ.NumField(); i++ {
		vItem := in.Field(i)
		switch vItem.Kind() {
			case reflect.Slice, reflect.Array, reflect.Map:
				item := vItem.Index(0)
				if _, ok := entities[item.Type().Name()]; ok {
					resources := getEmbeddedList(vItem, entities)
					emb[item.Type().Name()] = resources
				} else {
					resp[typ.Field(i).Name] = vItem.Interface()
				}
			default:
				if _, ok := entities[typ.Field(i).Name]; ok {
					resource := getResource(false, vItem, entities)
					emb[typ.Field(i).Name] = resource
				} else {
					resp[typ.Field(i).Name] = vItem.Interface()
				}
		}
	}
	resp["_embedded"] = emb
}

func getEmbeddedList(val reflect.Value, entities map[string]entity) []interface{} {
	embList := make([]interface{}, 0)

	for i := 0; i < val.Len(); i++ {
		vItem := val.Index(i)

		item := getResource(true, vItem, entities)

		embList = append(embList, item)
	}

	return embList
}

func getResource(embedded bool, in reflect.Value, entities map[string]entity) map[string]interface{} {
	resp := make(map[string]interface{}, 0)
	links := make(map[string]interface{}, 0)
	if subent, ok := entities[in.Type().Name()]; ok {
		// class := reflect.TypeOf(in).Name()
		halResourceLinks(links, subent, "", embedded)
	}
	if len(links) > 0 {
		resp["_links"] = links
	}

	typ := reflect.TypeOf(in.Interface())
	for i := 0; i < typ.NumField(); i++ {
		vItem := in.Field(i)
		resp[typ.Field(i).Name] = vItem.Interface()
	}

	return resp
}

// creates a new HAL Decorator 
func newHalDecorator() *Decorator {
	dec := Decorator{halDecorator}
	return &dec
}
