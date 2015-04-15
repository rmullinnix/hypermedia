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
func halDecorator(response interface{}, dec *Decorator) (interface{}) {
	var hm_resp 	HalDocument

	myDec = dec

	hm_resp = make(map[string]interface{}, 0)
	v := reflect.ValueOf(response)
	switch v.Kind() {
		case reflect.Struct:
			// Properties - not sub-entity items
			// Any sub-entities (struct or array), placed in Embedded
			class := reflect.TypeOf(response).Name()

			props, ents := stripEmbedded(v)

			links := halResourceLinks(myDec.GetEntity(class), props, false)
			curies := halDocumentCuries(myDec.GetEntity(class))

			for c_key, c_itm := range curies {
				links[c_key] = c_itm
			}

			hm_resp["_links"] = links
			for p_key, p_itm := range props {
				hm_resp[p_key] = p_itm
			}
			hm_resp["_embedded"] = ents
		case reflect.Slice, reflect.Array, reflect.Map:
			props := make(map[string]interface{})
			resources, class := getEmbeddedList(v)
			links := halResourceLinks(myDec.GetEntity(class), props, false)
			curies := halDocumentCuries(myDec.GetEntity(class))

			for c_key, c_itm := range curies {
				links[c_key] = c_itm
			}

			hm_resp["_links"] = links
			hm_resp["_embedded"] = resources
		default:
			hm_resp[reflect.TypeOf(response).Name()] = response
	}

	return hm_resp
}

func halResourceLinks(ent *entity, props map[string]interface{}, sub bool) (map[string]interface{}) {
	lnklist := make(map[string]interface{})

	for _, e_lnk := range ent.links {
		if sub && strings.IndexFunc(e_lnk.rel[:1], unicode.IsUpper) == 0 {
			continue
		}

		lnk := HalLink{e_lnk.href, e_lnk.templated, e_lnk.typ, "", e_lnk.name, "", e_lnk.title, ""}
		lnk.Href = myDec.UpdatePath(lnk.Href, props)
		lnklist[e_lnk.rel] = lnk
	}
	return lnklist
}

func halDocumentCuries(ent *entity) (map[string]interface{}) {
	lnklist := make(map[string]interface{})

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
	return lnklist
}

func stripEmbedded(in reflect.Value) (map[string]interface{}, map[string]interface{}) {
	emb := make(map[string]interface{})
	props := make(map[string]interface{})

	typ := reflect.TypeOf(in.Interface())
	for i := 0; i < typ.NumField(); i++ {
		vItem := in.Field(i)
		switch vItem.Kind() {
			case reflect.Slice, reflect.Array, reflect.Map:
				item := vItem.Index(0)
				if itm := myDec.GetEntity(item.Type().Name()); itm != nil {
					resources, _ := getEmbeddedList(vItem)
					emb[item.Type().Name()] = resources
				} else {
					props[typ.Field(i).Name] = vItem.Interface()
				}
			default:
				if itm := myDec.GetEntity(typ.Field(i).Name); itm != nil {
					resource := getEmbedded(false, vItem)
					emb[typ.Field(i).Name] = resource
				} else {
					props[typ.Field(i).Name] = vItem.Interface()
				}
		}
	}
	return props, emb
}

func getEmbeddedList(val reflect.Value) ([]interface{}, string) {
	var className		string

	embList := make([]interface{}, 0)

	for i := 0; i < val.Len(); i++ {
		vItem := val.Index(i)

		item := getEmbedded(true, vItem)

		embList = append(embList, item)

		if i == 0 {
			className = "[]" + vItem.Type().Name()
		}
	}

	return embList, className
}

func getEmbedded(embedded bool, in reflect.Value) map[string]interface{} {
	resp := make(map[string]interface{}, 0)
        typ := reflect.TypeOf(in.Interface())
	if subent := myDec.GetEntity(in.Type().Name()); subent != nil {
                val := reflect.ValueOf(in.Interface())
                for i := 0; i < typ.NumField(); i++ {
                        valf := val.Field(i)
                        resp[typ.Field(i).Name] = valf.Interface()
                }

		// class := reflect.TypeOf(in).Name()
		links := halResourceLinks(subent, resp, embedded)

		if len(links) > 0 {
			resp["_links"] = links
		}
	}

	return resp
}

// creates a new HAL Decorator 
func newHalDecorator() *indivDec {
	dec := new(indivDec)
	dec.Decorate = halDecorator
	return dec
}
