// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonschema

import (
	"fmt"
	"strings"
)

// InvalidJSONTypeError is the error type returned by ValidateInterface.
// this tells that specified go object is not valid jsonType.
type InvalidJSONTypeError string

func (e InvalidJSONTypeError) Error() string {
	return fmt.Sprintf("invalid jsonType: %s", string(e))
}

// InfiniteLoopError is returned by Compile.
// this gives keywordLocation that lead to infinity loop.
type InfiniteLoopError string

func (e InfiniteLoopError) Error() string {
	return "jsonschema: infinite loop " + string(e)
}

// SchemaError is the error type returned by Compile.
type SchemaError struct {
	// SchemaURL is the url to json-schema that filed to compile.
	// This is helpful, if your schema refers to external schemas
	SchemaURL string

	// Err is the error that occurred during compilation.
	// It could be ValidationError, because compilation validates
	// given schema against the json meta-schema
	Err error
}

func (se *SchemaError) Error() string {
	return fmt.Sprintf("json-schema %q compilation failed", se.SchemaURL)
}

func (se *SchemaError) GoString() string {
	if _, ok := se.Err.(*ValidationError); ok {
		return fmt.Sprintf("json-schema %q compilation failed. Reason:\n%#v", se.SchemaURL, se.Err)
	}
	return fmt.Sprintf("json-schema %q compilation failed. Reason: %v", se.SchemaURL, se.Err)
}

// ValidationError is the error type returned by Validate.
type ValidationError struct {
	// Message describes error
	Message string

	// InstancePtr is json-pointer which refers to json-fragment in json instance
	// that is not valid
	InstancePtr string

	// SchemaURL is the url to json-schema against which validation failed.
	// This is helpful, if your schema refers to external schemas
	SchemaURL string

	// SchemaPtr is json-pointer which refers to json-fragment in json schema
	// that failed to satisfy
	SchemaPtr string

	// Causes details the nested validation errors
	Causes []*ValidationError
}

func (ve *ValidationError) add(causes ...error) error {
	for _, cause := range causes {
		_ = addContext(ve.InstancePtr, ve.SchemaPtr, cause)
		ve.Causes = append(ve.Causes, cause.(*ValidationError))
	}
	return ve
}

// MessageFmt returns the Message formatted, but does not include child Cause messages.
//
// Deprecated: use Error method
func (ve *ValidationError) MessageFmt() string {
	return ve.Error()
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("I[%s] S[%s] %s", ve.InstancePtr, ve.SchemaPtr, ve.Message)
}

func (ve *ValidationError) GoString() string {
	msg := ve.Error()
	for _, c := range ve.Causes {
		for _, line := range strings.Split(c.GoString(), "\n") {
			msg += "\n  " + line
		}
	}
	return msg
}

func validationError(schemaPtr string, format string, a ...interface{}) *ValidationError {
	return &ValidationError{fmt.Sprintf(format, a...), "", "", schemaPtr, nil}
}

func addContext(instancePtr, schemaPtr string, err error) error {
	ve := err.(*ValidationError)
	ve.InstancePtr = joinPtr(instancePtr, ve.InstancePtr)
	if ve.SchemaURL == "" {
		ve.SchemaPtr = joinPtr(schemaPtr, ve.SchemaPtr)
	}
	for _, cause := range ve.Causes {
		_ = addContext(instancePtr, schemaPtr, cause)
	}
	return ve
}

func finishSchemaContext(err error, s *Schema) {
	ve := err.(*ValidationError)
	if len(ve.SchemaURL) == 0 {
		ve.SchemaURL = s.URL
		ve.SchemaPtr = joinPtr(s.Ptr, ve.SchemaPtr)
		for _, cause := range ve.Causes {
			finishSchemaContext(cause, s)
		}
	}
}

func finishInstanceContext(err error) {
	ve := err.(*ValidationError)
	ve.InstancePtr = absPtr(ve.InstancePtr)
	for _, cause := range ve.Causes {
		finishInstanceContext(cause)
	}
}

func joinPtr(ptr1, ptr2 string) string {
	if len(ptr1) == 0 {
		return ptr2
	}
	if len(ptr2) == 0 {
		return ptr1
	}
	return ptr1 + "/" + ptr2
}

func absPtr(ptr string) string {
	if ptr == "" {
		return "#"
	}
	if ptr[0] != '#' {
		return "#/" + ptr
	}
	return ptr
}
