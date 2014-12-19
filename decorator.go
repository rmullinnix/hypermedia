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
	"strings"
)

var decorators map[string]*Decorator

// siren - entity, hal - resource and embedded
type entity struct {
	class		string
	title		string
	typ		string
	href		string
	links		map[int]link
	actions		map[int]action
	curies		map[int]curie
}

// all hypermedia formats
type link struct {
	title		string
	class		string
	rel		string
	href		string
	typ		string
	templated	bool
	name		string
}

// siren - actions (leaving fields off for now)
type action struct {
	name		string
	method		string
	href		string
	class		string
	title		string
	typ		string
}

// hal - curie type, intended for documentation and URI prefix
type curie struct {
	name		string
	href		string
	templated	bool
}

type Entity bool		// all
type Link bool			// all
type Action bool		// siren
type Curie bool			// hal
//type Query bool			// collection+json
//type Error bool			// collection+json
//type Template bool		// collection+json
//type Data bool			// collection+json

var entityInitialized	bool
var entities 		map[string]entity
var srvr_prefix		string
var access		map[string]string
var security_enabled	bool

//Signiture of functions to be used as Decorators
type Decorator struct {
	Decorate func(string, interface{}, string) (interface{})
}

func NewHypermediaDecorator() {
	access = make(map[string]string)
	security_enabled = false
	registerHypermedia("application/vnd.siren+json", newSirenDecorator())
	registerHypermedia("application/hal+json", newHalDecorator())
}

func Decorate(mime string, prefix string, response interface{}, role string) (interface{}) {
	dec := getHypermedia(mime)
	if dec == nil {
		return response
	} else {
		return dec.Decorate(prefix, response, role)
	}
}

//Registers an Hypermedia Decorator for the specified mime type
func registerHypermedia(mime string, dec *Decorator) {
	if decorators == nil {
		decorators = make(map[string]*Decorator, 0)
	}
	if _, found := decorators[mime]; !found {
		decorators[mime] = dec
	}
}

//Returns the registred decorator for the specified mime type
func getHypermedia(mime string) (dec *Decorator) {
	if decorators == nil {
		decorators = make(map[string]*Decorator, 0)
	}
	dec, _ = decorators[mime]
	return
}

func RegisterEntity(i_ent interface{}) {

	if !entityInitialized {
		entities = make(map[string]entity)
		entityInitialized = true
	}

	t := reflect.TypeOf(i_ent)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	} else {
		panic("Invalid Interface")
	}

	if t.Kind() == reflect.Struct {
		if field, found := t.FieldByName("Entity"); found {
			temp := strings.Join(strings.Fields(string(field.Tag)), " ")
			ent := prepEntityData(reflect.StructTag(temp))
			ent.links = make(map[int]link)
			ent.actions = make(map[int]action)
			ent.curies = make(map[int]curie)

			linkcnt := 0
			actioncnt := 0
			curiecnt := 0
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				ftmp := strings.Join(strings.Fields(string(f.Tag)), " ")
				if f.Type.Name() == "Link" {
					lnk := prepLinkData(f.Name, reflect.StructTag(ftmp))
					ent.links[linkcnt] = lnk
					linkcnt++
				} else if f.Type.Name() == "Action" {
					act := prepActionData(f.Name, reflect.StructTag(ftmp))
					ent.actions[actioncnt] = act
					actioncnt++
				} else if f.Type.Name() == "Curie" {
					cur := prepCurieData(f.Name, reflect.StructTag(ftmp))
					ent.curies[curiecnt] = cur
					curiecnt++
				}
			}
			entities[ent.class] = ent	
		}
	}
}

func UnregisterEntity(classname string) {
	delete(entities, classname)
}

func prepEntityData(tags reflect.StructTag) entity {
	ent := new(entity)

	var tag		string

	if tag = tags.Get("class"); tag != "" {
		ent.class = tag
	}

	if tag = tags.Get("title"); tag != "" {
		ent.title = tag
	}

	if tag = tags.Get("href"); tag != "" {
		ent.href = tag
	}

	if tag = tags.Get("type"); tag != "" {
		ent.typ = tag
	}

	return *ent
}

func prepLinkData(rel string, tags reflect.StructTag) link {
	lnk := new(link)

	var tag		string

	lnk.rel = rel

	if tag = tags.Get("class"); tag != "" {
		lnk.class = tag
	}

	if tag = tags.Get("href"); tag != "" {
		lnk.href = tag
	}

	if tag = tags.Get("title"); tag != "" {
		lnk.title = tag
	}

	if tag = tags.Get("type"); tag != "" {
		lnk.typ = tag
	}

	return *lnk
}

func prepActionData(name string, tags reflect.StructTag) action {
	act := new(action)

	var tag		string

	act.name = name

	if tag = tags.Get("method"); tag != "" {
		act.method = tag
	}

	if tag = tags.Get("href"); tag != "" {
		act.href = tag
	}

	if tag = tags.Get("class"); tag != "" {
		act.class = tag
	}

	if tag = tags.Get("title"); tag != "" {
		act.title = tag
	}

	if tag = tags.Get("type"); tag != "" {
		act.typ = tag
	}

	return *act
}

func prepCurieData(name string, tags reflect.StructTag) curie {
	cur := new(curie)

	var tag		string

	cur.name = name

	if tag = tags.Get("href"); tag != "" {
		cur.href = tag
	}

	if tag = tags.Get("templated"); tag != "" {
		if tag == "true" {
			cur.templated = true
		} else if tag == "false" {
			cur.templated = false
		}
	}

	return *cur
}

func EnableSecurity() {
	security_enabled = true
}

func AddAccessRights(resource string, role string, rights string) {
	access[resource + role] = rights
}

func canAccessResource(resource string, role string, method string) bool {
	if !security_enabled {
		return true
	}

	chk := "other"
	switch method {
		case "GET": chk = "read"
		case "POST": chk = "create"
		case "PUT": chk = "update"
		case "DELETE": chk = "delete"
	}

	roleAccess, found := access[resource + role]
	if found {
		return strings.Contains(roleAccess, chk)
	}

	return false
}
