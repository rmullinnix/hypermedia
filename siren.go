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

type Siren struct {
	Class		string		`json:"class,omitempty"`
	Title		string		`json:"title,omitempty"`
	Properties	interface{}	`json:"properties"`
	Entities	[]SirenEntity	`json:"entities,omitempty"`
	Actions		[]SirenAction	`json:"actions,omitempty"`
	Links		[]SirenLink	`json:"links"`
}

type SirenEntity struct {
	Class		string		`json:"class,omitempty"`
	Rel		string		`json:"rel"`
	Properties	interface{}	`json:"properties"`
	Actions		[]SirenAction	`json:"actions,omitempty"`
	Links		[]SirenLink	`json:"links,omitempty"`
}

type SirenAction struct {
	Name		string		`json:"name"`
	Class		string		`json:"class,omitempty"`
	Method		string		`json:"method,omitempty"`
	Href		string		`json:"href"`
	Title		string		`json:"title,omitempty"`
	Type		string		`json:"type,omitempty"`
}

type Field struct {
	Name		string		`json:"name"`
	Type		string		`json:"type"`
	Value		string		`json:"value,omitempty"`
	Title		string		`json:"title,omitempty"`
}

type SirenLink struct {
	Class		string		`json:"class,omitempty"`
	Title		string		`json:"title,omitempty"`
	Rel		string		`json:"rel"`
	Href		string		`json:"href"`
	Type		string		`json:"type,omitempty"`
}

// This takes the data destined for the http response body and adds hypermedia content
// to the message prior to marshaling the data and returning it to the client
// The SirenDecorator loosely follows the siren specification
// mime type: applcation/vnd.siren+json 
func sirenDecorator(prefix string, response interface{}, role string) (interface{}) {
	var hm_resp 	Siren

	srvr_prefix = prefix

	v := reflect.ValueOf(response)
	switch v.Kind() {
		case reflect.Struct:
			// Properties - not sub-entity items
			// Any sub-entities (struct or array), placed in Entities
			props, ents := stripSubentities(v, entities, role)
			hm_resp.Properties = props
			hm_resp.Entities = ents
			hm_resp.Class = reflect.TypeOf(response).Name()
			hm_resp.Actions = sirenActions(entities[hm_resp.Class], props, role)
			hm_resp.Links = sirenLinks(entities[hm_resp.Class], props, role)
		case reflect.Slice, reflect.Array, reflect.Map:
			props := make(map[string]interface{})
			hm_resp.Entities, hm_resp.Class = getEntityList(v, entities, role)
			hm_resp.Actions = sirenActions(entities[hm_resp.Class], props, role)
			hm_resp.Links = sirenLinks(entities[hm_resp.Class], props, role)
		default:
			hm_resp.Properties = response
			hm_resp.Class = reflect.TypeOf(response).Name()
	}

	return hm_resp
}

func sirenLinks(ent entity, props map[string]interface{}, role string) []SirenLink {
	lnklist := make([]SirenLink, 0)
	
	for _, e_lnk := range ent.links {
		if canAccessResource(e_lnk.class, role, "GET") {
			lnk := SirenLink{"", "", e_lnk.rel, e_lnk.href, ""}
			lnk.Href = updatePath(lnk.Href, props)
			lnklist = append(lnklist, lnk)
		}
	}
	return lnklist
}

func sirenActions(ent entity, props map[string]interface{}, role string) []SirenAction {
	actlist := make([]SirenAction, 0)
	
	for _, e_act := range ent.actions {
		if canAccessResource(e_act.class, role, e_act.method) {
			act := SirenAction{e_act.name, e_act.class, e_act.method, e_act.href, "", ""}
			act.Href = updatePath(act.Href, props)
			actlist = append(actlist, act)
		}
	}
	return actlist
}

func stripSubentities(in reflect.Value, entities map[string]entity, role string) (map[string]interface{}, []SirenEntity) {
	ents :=	[]SirenEntity{}
	out := make(map[string]interface{}, 30)

	typ := reflect.TypeOf(in.Interface())
	for i := 0; i < typ.NumField(); i++ {
		vItem := in.Field(i)
		switch vItem.Kind() {
			case reflect.Slice, reflect.Array, reflect.Map:
				if vItem.Len() > 0 {
					item := vItem.Index(0)
					if _, ok := entities[item.Type().Name()]; ok {
						tmp, _ := getEntityList(vItem, entities, role)
						ents = append(ents, tmp...)
					} else {
						out[typ.Field(i).Name] = vItem.Interface()
					}
				} else {
					out[typ.Field(i).Name] = vItem.Interface()
				}
			default:
				if _, ok := entities[typ.Field(i).Name]; ok {
					item := getEntity(false, vItem, entities, role)
					ents = append(ents, item)
				} else {
					out[typ.Field(i).Name] = vItem.Interface()
				}
		}
	}

	return out, ents
}

func getEntityList(val reflect.Value, entities map[string]entity, role string) ([]SirenEntity, string) {
	entList := []SirenEntity{}
	var className string

	for i := 0; i < val.Len(); i++ {
		vItem := val.Index(i)

		item := getEntity(true, vItem, entities, role)
		item.Class = vItem.Type().Name() + " list-item"

		entList = append(entList, item)

		if i == 0 {
			className = "[]" + vItem.Type().Name()
		}
	}

	return entList, className
}

func getEntity(sub bool, vItem reflect.Value, entities map[string]entity, role string) SirenEntity {
	var item	SirenEntity

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

			if canAccessResource(subent.links[j].class, role, "GET") {
				lnk := SirenLink{"", "", subent.links[j].rel, subent.links[j].href, ""}

				lnk.Href = updatePath(lnk.Href, props)

				item.Links = append(item.Links, lnk)
			}
		}

		for j:= 0; j < len(subent.actions); j++ {
			if canAccessResource(subent.actions[j].class, role, subent.actions[j].method) {
				act := SirenAction{subent.actions[j].name, subent.actions[j].class, subent.actions[j].method, subent.actions[j].href, "", ""}

				act.Href = updatePath(act.Href, props)

				item.Actions = append(item.Actions, act)
			}
		}
	}
	return item
}

func updatePath(path string, props map[string]interface{}) string {

	reg := regexp.MustCompile("{[^}]+}")
	parts := reg.FindAllString(path, -1)

	for _, str1 := range parts {
		if strings.HasPrefix(str1, "{") && strings.HasSuffix(str1, "}") {
			
			str2 := str1[1:len(str1) - 1]
			if pos := strings.IndexAny(str2, "+-"); pos > -1  {
				if item, found := props[str2[:pos]]; found {
					value := reflect.ValueOf(item).Int()
					val2, _ := strconv.Atoi(str2[pos+1:])
					if strings.Contains(str2, "+")  {
						value = value + int64(val2)
					} else {
						value = value - int64(val2)
					}
					path = strings.Replace(path, str1, strconv.FormatInt(value, 10), 1)
				}
			} else if item, found := props[str2]; found {
				path = strings.Replace(path, str1, getValueString(item), 1)
			}
		}
	}
	path = srvr_prefix + path

	return path
}

func getValueString(item interface{}) string {
	value := "<invalid>"
	vItem := reflect.ValueOf(item)
	switch vItem.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			value = strconv.FormatInt(vItem.Int(), 10)
		case reflect.Bool:
			value = strconv.FormatBool(vItem.Bool())
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			value = strconv.FormatUint(vItem.Uint(), 10)
		case reflect.Float32:
			value = strconv.FormatFloat(vItem.Float(), 'e', -1, 32)
		case reflect.Float64:
			value = strconv.FormatFloat(vItem.Float(), 'e', -1, 64)
		case reflect.String:
			value = vItem.String()
	}
	return value
}

// creates a new Siren Decorator 
func newSirenDecorator() *Decorator {
	dec := Decorator{sirenDecorator}
	return &dec
}
