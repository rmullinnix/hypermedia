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
	"reflect"
	"strconv"
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

var myDec *Decorator

// This takes the data destined for the http response body and adds hypermedia content
// to the message prior to marshaling the data and returning it to the client
// The SirenDecorator loosely follows the siren specification
// mime type: applcation/vnd.siren+json 
func sirenDecorator(response interface{}, dec *Decorator) (interface{}) {
	var hm_resp 	Siren
	myDec = dec

	v := reflect.ValueOf(response)
	switch v.Kind() {
		case reflect.Struct:
			// Properties - not sub-entity items
			// Any sub-entities (struct or array), placed in Entities
			props, ents := stripSubentities(v)
			hm_resp.Properties = props
			hm_resp.Entities = ents
			hm_resp.Class = reflect.TypeOf(response).Name()
			hm_resp.Actions = sirenActions(myDec.GetEntity(hm_resp.Class), props)
			hm_resp.Links = sirenLinks(myDec.GetEntity(hm_resp.Class), props)
		case reflect.Slice, reflect.Array, reflect.Map:
			props := make(map[string]interface{})
			hm_resp.Entities, hm_resp.Class = getEntityList(v)
			hm_resp.Actions = sirenActions(myDec.GetEntity(hm_resp.Class), props)
			hm_resp.Links = sirenLinks(myDec.GetEntity(hm_resp.Class), props)
		default:
			hm_resp.Properties = response
			hm_resp.Class = reflect.TypeOf(response).Name()
	}

	return hm_resp
}

func sirenLinks(ent *entity, props map[string]interface{}) []SirenLink {
	lnklist := make([]SirenLink, 0)
	
	if ent != nil {
		for _, e_lnk := range ent.links {
			if myDec.hasAccess(e_lnk.href, "GET") {
				lnk := SirenLink{"", "", e_lnk.rel, e_lnk.href, ""}
				lnk.Href = myDec.UpdatePath(lnk.Href, props)
				lnklist = append(lnklist, lnk)
			}
		}
	}
	return lnklist
}

func sirenActions(ent *entity, props map[string]interface{}) []SirenAction {
	actlist := make([]SirenAction, 0)
	
	if ent != nil {
		for _, e_act := range ent.actions {
			if myDec.hasAccess(e_act.href, e_act.method) {
				act := SirenAction{e_act.name, e_act.class, e_act.method, e_act.href, "", ""}
				act.Href = myDec.UpdatePath(act.Href, props)
				actlist = append(actlist, act)
			}
		}
	}
	return actlist
}

func stripSubentities(in reflect.Value) (map[string]interface{}, []SirenEntity) {
	ents :=	[]SirenEntity{}
	out := make(map[string]interface{}, 30)

	typ := reflect.TypeOf(in.Interface())
	for i := 0; i < typ.NumField(); i++ {
		vItem := in.Field(i)
		switch vItem.Kind() {
			case reflect.Slice, reflect.Array, reflect.Map:
				if vItem.Len() > 0 {
					item := vItem.Index(0)
					if itm := myDec.GetEntity(item.Type().Name()); itm != nil {
						tmp, _ := getEntityList(vItem)
						ents = append(ents, tmp...)
					} else {
						out[typ.Field(i).Name] = vItem.Interface()
					}
				} else {
					out[typ.Field(i).Name] = vItem.Interface()
				}
			default:
				if itm := myDec.GetEntity(typ.Field(i).Name); itm != nil {
					item := getEntity(false, vItem, "class")
					ents = append(ents, item)
				} else {
					out[typ.Field(i).Name] = vItem.Interface()
				}
		}
	}

	return out, ents
}

func getEntityList(val reflect.Value) ([]SirenEntity, string) {
	entList := []SirenEntity{}
	var className string

	for i := 0; i < val.Len(); i++ {
		vItem := val.Index(i)

		item := getEntity(true, vItem, "list")
		item.Class = vItem.Type().Name() + " list-item"

		entList = append(entList, item)

		if i == 0 {
			className = "[]" + vItem.Type().Name()
		}
	}

	return entList, className
}

func getEntity(sub bool, vItem reflect.Value, colType string) SirenEntity {
	var item	SirenEntity

	item.Class = vItem.Type().Name()
	item.Rel = vItem.Type().Name()
	item.Properties = vItem.Interface()

	if subent := myDec.GetEntity(vItem.Type().Name()); subent != nil {
		typ := reflect.TypeOf(item.Properties)
		val := reflect.ValueOf(item.Properties)
		props := make(map[string]interface{}, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			valf := val.Field(i)
			props[typ.Field(i).Name] = valf.Interface()
		}

		for j:= 0; j < len(subent.links); j++ {
			process := false
			if (subent.links[j].in == "both" || subent.links[j].in == "list") && colType == "list" {
				process = true
			}

			if (subent.links[j].in == "both" || subent.links[j].in == "class") && colType == "class" {
				process = true
			}

			if process {
				if myDec.hasAccess(subent.links[j].href, "GET") {
					lnk := SirenLink{"", "", subent.links[j].rel, subent.links[j].href, ""}

					lnk.Href = myDec.UpdatePath(lnk.Href, props)

					item.Links = append(item.Links, lnk)
				}
			}
		}

		for j:= 0; j < len(subent.actions); j++ {
			process := false
			if (subent.actions[j].in == "both" || subent.actions[j].in == "list") && colType == "list" {
				process = true
			}

			if (subent.actions[j].in == "both" || subent.actions[j].in == "class") && colType == "class" {
				process = true
			}

			if process {
				if myDec.hasAccess(subent.actions[j].href, subent.actions[j].method) {
					act := SirenAction{subent.actions[j].name, subent.actions[j].class, subent.actions[j].method, subent.actions[j].href, "", ""}

					act.Href = myDec.UpdatePath(act.Href, props)

					item.Actions = append(item.Actions, act)
				}
			}
		}
	}
	return item
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
func newSirenDecorator() *indivDec {
	dec := new(indivDec)
	dec.Decorate = sirenDecorator
	return dec
}
