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
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type HypermediaDef struct {
	Resources       map[string]ResourceDef  `json:"resources"`
	Classes         map[string]ClassDef     `json:"classes"`
}

type ResourceDef struct {
	Href            string                  `json:"href"`
	Version         string                  `json:"version"`
}

type ClassDef struct {
	ResourceName    string                  `json:"resource"`
	Actions         []ActionDef             `json:"actions"`
	Links           []LinkDef               `json:"links"`
}

type ActionDef struct {
	Name            string                  `json:"name"`
	Class           string                  `json:"class"`
	Method          string                  `json:"method"`
	Href            string                  `json:"href"`
	In		string			`json:"in"`
}

type LinkDef struct {
	Name            string                  `json:"name"`
	Class           string                  `json:"class"`
	Href            string                  `json:"href"`
	In		string			`json:"in"`
}

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
	in		string
}

// siren - actions (leaving fields off for now)
type action struct {
	name		string
	method		string
	href		string
	class		string
	title		string
	typ		string
	in		string
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

type Decorator struct {
	decorators		map[string]*indivDec
	entities 		map[string]entity
	scopes			map[string]bool
	paths			map[string][]string
	srvr_prefix		string
	security_enabled	bool
}

//Signiture of functions to be used as Decorators
type indivDec struct {
	Decorate func(interface{}, *Decorator) (interface{})
}

func NewHypermediaDecorator() Decorator {
	dec := new(Decorator)
	dec.decorators = make(map[string]*indivDec, 0)

	dec.security_enabled = false
	dec.registerHypermedia("application/vnd.siren+json", newSirenDecorator())
	dec.registerHypermedia("application/hal+json", newHalDecorator())
	dec.entities = make(map[string]entity)
	dec.paths = make(map[string][]string)

	return *dec
}

func (this Decorator) Decorate(mime string, prefix string, response interface{}, scopes []string) (interface{}) {
	dec := this.getHypermedia(mime)
	if dec == nil {
		return response
	} else {
		this.srvr_prefix = prefix
		this.scopes = make(map[string]bool)
		for i := range scopes {
			strScope := scopes[i]
			hasContext := false
			if pos := strings.Index(scopes[i], "["); pos > -1 {
				strScope = strScope[:pos]
				hasContext = true
			}
			this.scopes[strScope] = hasContext
		}
		return dec.Decorate(response, &this)
	}
}

//Registers an Hypermedia Decorator for the specified mime type
func (this Decorator) registerHypermedia(mime string, dec *indivDec) {
	if _, found := this.decorators[mime]; !found {
		this.decorators[mime] = dec
	}
}

//Returns the registred decorator for the specified mime type
func (this Decorator) getHypermedia(mime string) (dec *indivDec) {
	dec, _ = this.decorators[mime]
	return
}

func (this Decorator) RegisterDefinition(hmDef HypermediaDef) {
	for className, classData := range hmDef.Classes {
		var ent		entity

		ent.links = make(map[int]link)
		ent.actions = make(map[int]action)
		ent.curies = make(map[int]curie)

		ent.class = className
		ent.href = hmDef.Resources[classData.ResourceName].Href

		for i := range classData.Actions {
			var newAction	action

			newAction.name = classData.Actions[i].Name
			newAction.method = classData.Actions[i].Method
			newAction.href = hmDef.Resources[classData.Actions[i].Class].Href + classData.Actions[i].Href
			newAction.class = classData.Actions[i].Class
			newAction.in = classData.Actions[i].In

			ent.actions[i] = newAction
		}

		for i := range classData.Links {
			var newLink	link

			newLink.name = classData.Links[i].Name
			newLink.rel = classData.Links[i].Name
			newLink.href = hmDef.Resources[classData.Links[i].Class].Href + classData.Links[i].Href
			newLink.class = classData.Links[i].Class
			newLink.in = classData.Links[i].In

			ent.links[i] = newLink
		}

		this.entities[className] = ent
	}
}
 
func (this Decorator) RegisterEntity(i_ent interface{}) {
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
			this.entities[ent.class] = ent	
		}
	}
}

func (this Decorator) UnregisterEntity(classname string) {
	delete(this.entities, classname)
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

func (this Decorator) EnableSecurity() {
	this.security_enabled = true
}

func (this Decorator) UpdatePath(path string, props map[string]interface{}) string {

	reg := regexp.MustCompile("{[^}]+}")
	parts := reg.FindAllString(path, -1)

	for _, str1 := range parts {
		if strings.HasPrefix(str1, "{") && strings.HasSuffix(str1, "}") {
			
			str2 := str1[1:len(str1) - 1]
			if item, found := props[str2]; found {
				path = strings.Replace(path, str1, getValueString(item), 1)
			}
		}
	}
	path = this.srvr_prefix + "/" + path

	return path
}

func (this Decorator) GetEntity(entName string) *entity {
	ent, found := this.entities[entName]
	if found {
		return &ent
	} else {
		return nil
	}
}

func (this Decorator) AddAccess(path string, method string, scope []string) {
	methPath := method + ":" + path
	this.paths[methPath] = make([]string, 0)
	this.paths[methPath] = append(this.paths[methPath], scope...)
}

func (this Decorator) hasAccess(path string, method string) bool {
	fmt.Println("check access", path, method)
	methPath := method + ":" + path
	access, found := this.paths[methPath]
	if found {
		for i := range access {
			if _, hasScope := this.scopes[access[i]]; hasScope {
				fmt.Println("  scope found")
				return true
			} else if access[i] == "<valid>" {
				return true
			}
		}
	} else {
		fmt.Println("  Path not found")
		return true
	}
	fmt.Println("  Scope not found")
	return false
}
