package main

import "net/http"
import "fmt"
import "reflect"
import "strconv"
import "encoding/json"

type Response struct {
	Err  string      `json:"error"`
	Resp interface{} `json:"response,omitempty"`
}

func CompStrSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MyApi
func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {

	case "/user/profile":
		h.wrapperProfile(w, r)

	case "/user/create":
		h.wrapperCreate(w, r)

	default:
		response := Response{}
		w.WriteHeader(http.StatusNotFound)
		response.Err = "unknown method"
		resp, _ := json.Marshal(response)
		fmt.Fprintf(w, string(resp))
	}
}

func (h *MyApi) wrapperProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	response := Response{}

	params := ProfileParams{

		Login: r.FormValue("login"),
	}

	if params.Login == "" {

		response.Err = "login must me not empty"

		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	res, err := h.Profile(ctx, params)
	if err != nil {
		response.Err = err.Error()
		if reflect.TypeOf(err).Name() == "ApiError" {
			w.WriteHeader(err.(ApiError).HTTPStatus)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		resp, _ := json.Marshal(response)
		//w.WriteHeader(err.(*ApiError).HTTPStatus)
		fmt.Fprintf(w, string(resp))
		return
	}
	response.Resp = res
	response.Err = ""
	resp, _ := json.Marshal(response)
	fmt.Fprintf(w, string(resp))
}

func (h *MyApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	response := Response{}

	authKey := []string{"100500"}
	if !CompStrSlices(r.Header["X-Auth"], authKey) {
		w.WriteHeader(http.StatusForbidden)
		response.Err = "unauthorized"
		resp, _ := json.Marshal(response)
		fmt.Fprintf(w, string(resp))
		return
	}

	if r.Method != "POST" {
		response.Err = "bad method"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, string(resp))
		return
	}

	params := CreateParams{

		Login: r.FormValue("login"),

		Name: r.FormValue("full_name"),

		Status: r.FormValue("status"),
	}

	age, errI := strconv.Atoi(r.FormValue("age"))
	if errI != nil {
		response.Err = "age must be int"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}
	params.Age = age

	if params.Login == "" {

		response.Err = "login must me not empty"

		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if len(params.Login) < 10 {
		response.Err = "login len must be >= 10"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if params.Status == "" {
		params.Status = "user"
	}

	if params.Status != "user" && params.Status != "moderator" && params.Status != "admin" {
		response.Err = "status must be one of [user, moderator, admin]"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if params.Age < 0 {
		response.Err = "age must be >= 0"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if params.Age > 128 {
		response.Err = "age must be <= 128"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	res, err := h.Create(ctx, params)
	if err != nil {
		response.Err = err.Error()
		if reflect.TypeOf(err).Name() == "ApiError" {
			w.WriteHeader(err.(ApiError).HTTPStatus)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		resp, _ := json.Marshal(response)
		//w.WriteHeader(err.(*ApiError).HTTPStatus)
		fmt.Fprintf(w, string(resp))
		return
	}
	response.Resp = res
	response.Err = ""
	resp, _ := json.Marshal(response)
	fmt.Fprintf(w, string(resp))
}

// OtherApi
func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {

	case "/user/create":
		h.wrapperCreate(w, r)

	default:
		response := Response{}
		w.WriteHeader(http.StatusNotFound)
		response.Err = "unknown method"
		resp, _ := json.Marshal(response)
		fmt.Fprintf(w, string(resp))
	}
}

func (h *OtherApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	response := Response{}

	authKey := []string{"100500"}
	if !CompStrSlices(r.Header["X-Auth"], authKey) {
		w.WriteHeader(http.StatusForbidden)
		response.Err = "unauthorized"
		resp, _ := json.Marshal(response)
		fmt.Fprintf(w, string(resp))
		return
	}

	if r.Method != "POST" {
		response.Err = "bad method"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, string(resp))
		return
	}

	params := OtherCreateParams{

		Username: r.FormValue("username"),

		Name: r.FormValue("account_name"),

		Class: r.FormValue("class"),
	}

	level, errI := strconv.Atoi(r.FormValue("level"))
	if errI != nil {
		response.Err = "level must be int"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}
	params.Level = level

	if params.Username == "" {

		response.Err = "username must me not empty"

		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if len(params.Username) < 3 {
		response.Err = "username len must be >= 3"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if params.Class == "" {
		params.Class = "warrior"
	}

	if params.Class != "warrior" && params.Class != "sorcerer" && params.Class != "rouge" {
		response.Err = "class must be one of [warrior, sorcerer, rouge]"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if params.Level < 1 {
		response.Err = "level must be >= 1"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	if params.Level > 50 {
		response.Err = "level must be <= 50"
		resp, _ := json.Marshal(response)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, string(resp))
		return
	}

	res, err := h.Create(ctx, params)
	if err != nil {
		response.Err = err.Error()
		if reflect.TypeOf(err).Name() == "ApiError" {
			w.WriteHeader(err.(ApiError).HTTPStatus)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		resp, _ := json.Marshal(response)
		//w.WriteHeader(err.(*ApiError).HTTPStatus)
		fmt.Fprintf(w, string(resp))
		return
	}
	response.Resp = res
	response.Err = ""
	resp, _ := json.Marshal(response)
	fmt.Fprintf(w, string(resp))
}
