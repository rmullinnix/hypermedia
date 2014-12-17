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
	"regexp"
	"strconv"
//	"unicode"
)

type CJDoc struct {
	Collection	CJCollection	`json:"collection"`
}

type CJCollection struct {
	Version		string		`json:"version"`
	Href		string		`json:"href"`
	Links		[]CJLink	`json:"links,omitempty"`
	Items		[]CJItem	`json:"items,omitempty"`
	Queries		[]CJQuery	`json:"queries,omitempty"`
	Template	CJTemplate	`json:"template,omitempty"`
	Error		CJError		`json:"error,omitempty"`
}

type CJLink struct {
	Href		string		`json:"href"`
	Rel		string		`json:"rel"`
	Prompt		string		`json:"prompt,omitempty"`
	Name		string		`json:"name",omitempty"`
	Render		string		`json:"render,omitempty"`
}

type CJItem struct {
	Href		string		`json:"href"`
	Data		[]CJData	`json:"data,omitempty"`
	Links		[]CJLink	`json:"links,omitempty"`
}

type CJQuery struct {
	Href		string		`json:"href"`
	Rel		string		`json:"rel"`
	Prompt		string		`json:"prompt,omitempty"`
	Name		string		`json:"name",omitempty"`
	Data		[]CJData	`json:"data,omitempty"`
}

type CJTemplate struct {
	Data		[]CJData	 `json:"data,omitempty"`
}

type CJData struct {
	Prompt		string		`json:"prompt,omitempty"`
	Name		string		`json:"name"`
	Value		interface{}	`json:"value",omitempty"`
}

type CJError struct {
	Title		string		`json:"title,omitmepty"`
	Code		string		`json:"code,omitmepty"`
	Message		string		`json:"message,omitmepty"`
}

// This takes the data destined for the http response body and adds hypermedia content
// to the message prior to marshaling the data and returning it to the client
// The CollectionDecorator loosely follows the collection+json specification
// mime type: applcation/vnd.collection+json 
func collectionDecorator(prefix string, response interface{}) (interface{}) {
	var hm_resp 	CJDoc

	srvr_prefix = prefix

	v := reflect.ValueOf(response)
	switch v.Kind() {
		case reflect.Struct:
			// Properties - not sub-entity items
			// Any sub-entities (struct or array), placed in Entities
			props, items := stripSubItems(v, entities)
			hm_resp.Properties = props
			hm_resp.Collection.Items = items
			hm_resp.Class = reflect.TypeOf(response).Name()
			hm_resp.Links = collectionLinks(entities[hm_resp.Class], props)
		case reflect.Slice, reflect.Array, reflect.Map:
			props := make(map[string]interface{})
			hm_resp.Entities, hm_resp.Class = getEntityList(v, entities)
			hm_resp.Actions = collectionActions(entities[hm_resp.Class], props)
			hm_resp.Links = collectionLinks(entities[hm_resp.Class], props)
		default:
			hm_resp.Properties = response
			hm_resp.Class = reflect.TypeOf(response).Name()
	}

	return hm_resp
}

func collectionLinks(ent entity, props map[string]interface{}) []CJLink {
	lnklist := make([]CJLink, len(ent.links))
	i := 0
	
	for _, e_lnk := range ent.links {
		lnk := CJLink{e_lnk.href, e_lnk.rel, "", "", ""}
		lnk.Href = updatePath(lnk.Href, props)
		lnklist[i] = lnk
		i++
	}
	return lnklist
}

func stripSubItems(in reflect.Value, entities map[string]entity) (map[string]interface{}, []CollectionEntity) {
	ents :=	[]CollectionEntity{}
	out := make(map[string]interface{}, 30)

	typ := reflect.TypeOf(in.Interface())
	for i := 0; i < typ.NumField(); i++ {
		vItem := in.Field(i)
		switch vItem.Kind() {
			case reflect.Slice, reflect.Array, reflect.Map:
				if vItem.Len() > 0 {
					item := vItem.Index(0)
					if _, ok := entities[item.Type().Name()]; ok {
						tmp, _ := getEntityList(vItem, entities)
						ents = append(ents, tmp...)
					} else {
						out[typ.Field(i).Name] = vItem.Interface()
					}
				} else {
					out[typ.Field(i).Name] = vItem.Interface()
				}
			default:
				if _, ok := entities[typ.Field(i).Name]; ok {
					item := getEntity(false, vItem, entities)
					ents = append(ents, item)
				} else {
					out[typ.Field(i).Name] = vItem.Interface()
				}
		}
	}

	return out, ents
}

func getEntityList(val reflect.Value, entities map[string]entity) ([]CollectionEntity, string) {
	entList := []CollectionEntity{}
	var className string

	for i := 0; i < val.Len(); i++ {
		vItem := val.Index(i)

		item := getEntity(true, vItem, entities)
		item.Class = vItem.Type().Name() + " list-item"

		entList = append(entList, item)

		if i == 0 {
			className = "[]" + vItem.Type().Name()
		}
	}

	return entList, className
}

func getEntity(sub bool, vItem reflect.Value, entities map[string]entity) CollectionEntity {
	var item	CollectionEntity

	item.Class = vItem.Type().Name()
	item.Rel = vItem.Type().Name()
	item.Properties = vItem.Interface()

	if subent, ok := entities[vItem.Type().Name()]; ok {
		typ := reflect.TypeOf(item.Properties)
		val := reflect.ValueOf(item.Properties)
		props := make(map[string]interface{}, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			valf := val.Field(i)
			props[typ.Field(i).Name] = valf.Interface()
		}

		for j:= 0; j < len(subent.links); j++ {
	//		if sub && strings.IndexFunc(subent.links[j].rel[:1], unicode.IsUpper) == 0 {
	//			continue
	//		}

			lnk := CJLink{"", "", subent.links[j].rel, subent.links[j].href, ""}

			lnk.Href = updatePath(lnk.Href, props)

			item.Links = append(item.Links, lnk)
		}

		for j:= 0; j < len(subent.actions); j++ {
			act := CollectionAction{subent.actions[j].name, subent.actions[j].class, subent.actions[j].method, subent.actions[j].href, "", ""}

			act.Href = updatePath(act.Href, props)

			item.Actions = append(item.Actions, act)
		}
	}
	return item
}

// creates a new Collection Decorator 
func newCollectionDecorator() *Decorator {
	dec := Decorator{collectionDecorator}
	return &dec
}
