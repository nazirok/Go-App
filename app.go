package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const requestsFileName = "list_responses.json"

func uuid() string {
	/*
		Генератор уникальных id
	*/

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}

type JsonRequest struct {
	method  string
	url     string
	headers string
	// body []uint8 // вернуть потом обратно
	body string
}

func addInfo(data map[string]any) {
	/*
		Добавляет request+response в JSON файл.
	*/

	fileRead, _ := ioutil.ReadFile(requestsFileName)
	var jsonStruct = readAndReturn()
	err := json.Unmarshal(fileRead, &jsonStruct)
	if err != nil {
		panic(err)
	}
	var linkToResponses = jsonStruct["responses"]
	linkToResponses = append(linkToResponses, data)
	jsonStruct["responses"] = linkToResponses
	var dataToAdd, _ = json.MarshalIndent(jsonStruct, "", "    ")
	err2 := ioutil.WriteFile(requestsFileName, dataToAdd, 0600)
	if err != nil {
		panic(err2)
	}

}
func removeInfo(id string) bool {

	var removed = false
	fileRead, _ := ioutil.ReadFile(requestsFileName)
	var jsonStruct = readAndReturn()
	err := json.Unmarshal(fileRead, &jsonStruct)
	if err != nil {
		panic(err)
	}
	var linkToResponses = jsonStruct["responses"]
	var newResponses []any
	for _, value := range linkToResponses {
		var valueMap = value.(map[string]any)
		if (valueMap["id"]).(string) != id {
			newResponses = append(newResponses, value)
		} else if (valueMap["id"]).(string) == id {
			removed = true

		}
	}
	jsonStruct["responses"] = newResponses
	var dataToAdd, _ = json.MarshalIndent(jsonStruct, "", "    ")
	err2 := ioutil.WriteFile(requestsFileName, dataToAdd, 0600)
	if err2 != nil {
		panic(err2)
	}

	return removed

}
func readAndReturn() map[string][]any {
	/*
		Читает JSON файл и возвращает его данные в GO формате.
	*/

	fileRead, _ := ioutil.ReadFile(requestsFileName)
	var jsonStruct = map[string][]any{}
	err := json.Unmarshal(fileRead, &jsonStruct)
	if err != nil {
		panic(err)
	}

	return jsonStruct

}

func GetMD5Hash(text string) string {
	/*
		Генерирует хэш из строки
	*/

	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func searchById(id string) []byte {
	/*
		Делает поиск внутри файла JSON по id, если id совпадают, возвращает сохраненный с ним request+response
	*/

	var jsonStruct = readAndReturn()
	var responses = jsonStruct["responses"]

	for _, interfaceIter := range responses {
		var interfaceMap = interfaceIter.(map[string]any)
		if interfaceMap["id"] == id {
			data, _ := json.MarshalIndent(interfaceIter, "", "    ")
			return data
		}

	}
	return nil

}
func CacheLRU(request JsonRequest) []byte {
	var body = request.body
	var method = request.method
	var headers = request.headers
	var url = request.url
	var jsonStruct = readAndReturn()
	var responses = jsonStruct["responses"]

	for _, interfaceIter := range responses {
		var interfaceMap = interfaceIter.(map[string]any)
		requestLocal := interfaceMap["request"]
		responseLocal := interfaceMap["response"]
		var methodLocal = (requestLocal.(map[string]any)["method"]).(string)
		var urlLocal = (requestLocal.(map[string]any)["url"]).(string)
		headerstLocal := requestLocal.(map[string]any)["headers"]
		bodyLocal := requestLocal.(map[string]any)["body"]
		var headerstLocalString = fmt.Sprintf("%s", headerstLocal)
		var bodyLocalString = fmt.Sprintf("%s", bodyLocal)

		if methodLocal == method {
			if urlLocal == url {
				if GetMD5Hash(headerstLocalString) == GetMD5Hash(headers) {
					if GetMD5Hash(bodyLocalString) == GetMD5Hash(body) {
						data, _ := json.MarshalIndent(responseLocal, "", "    ")
						return data

					} else if len(body) == 0 && bodyLocal == nil {
						data, _ := json.MarshalIndent(responseLocal, "", "    ")
						return data

					}

				} else if len(headers) == 0 && headerstLocal == nil {
					if GetMD5Hash(bodyLocalString) == GetMD5Hash(body) {
						data, _ := json.MarshalIndent(responseLocal, "", "    ")
						return data

					} else if len(body) == 0 && bodyLocal == nil {
						data, _ := json.MarshalIndent(responseLocal, "", "    ")
						return data

					}

				}
			}

		}

	}
	return nil

}

func main() {

	var server = func(w http.ResponseWriter, r *http.Request) {
		decoder, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		var jsonReq = map[string]any{}
		err2 := json.Unmarshal(decoder, &jsonReq)
		if err != nil {
			panic(err2)
		}
		if r.Method == http.MethodPost {
			var body map[string]interface{}
			if jsonReq["body"] == nil {
				body = nil
			} else {
				body = (jsonReq["body"]).(map[string]interface{})
			}

			var headers map[string]interface{}
			if jsonReq["headers"] == nil {
				headers = nil
			} else {
				headers = (jsonReq["headers"]).(map[string]interface{})
			}

			var method = (jsonReq["method"]).(string)
			var url = (jsonReq["url"]).(string)

			var headersString string
			var bodyString string
			if body != nil {
				bodyString = fmt.Sprintf("%s", body)
			} else if body == nil {
				bodyString = ""
			}
			if headers != nil {
				headersString = fmt.Sprintf("%s", headers)
			} else if headers == nil {
				headersString = ""
			}

			var jsR = JsonRequest{method: method, url: url, headers: headersString, body: bodyString}
			var cacheLRU = CacheLRU(jsR)
			if cacheLRU == nil {
				var httpResponse = HttpRequest(method, url, headers, body)
				var request = map[string]any{}
				var contentType = httpResponse.Header["Content-Type"][0]
				var secondHeaders = map[string]any{"Content-Length": httpResponse.ContentLength, "Content-Type": contentType}
				if headersString == "" && bodyString == "" {
					request = map[string]any{"method": method, "url": url}

				} else if headersString == "" {
					request = map[string]any{"method": method, "url": url, "body": body}

				} else if bodyString == "" {
					request = map[string]any{"method": method, "url": url, "headers": headers}

				} else {
					request = map[string]any{"method": method, "url": url, "headers": headers, "body": body}
				}
				var response = map[string]any{"status": httpResponse.StatusCode, "headers": secondHeaders, "length": httpResponse.ContentLength}
				var headersData = map[string]any{"id": uuid(), "request": request, "response": response}
				addInfo(headersData)
				var dataToWatch, _ = json.MarshalIndent(response, "", "    ")
				_, err := w.Write(dataToWatch)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)

			} else {
				_, err2 := w.Write(cacheLRU)
				if err2 != nil {
					panic(err2)
				}
				w.WriteHeader(200)
			}

		}
		if r.Method == http.MethodGet {
			if jsonReq["id"] == nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			} else {
				var text = searchById(jsonReq["id"].(string))
				if text == nil {
					http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				} else {
					_, err := w.Write(text)
					if err != nil {
						panic(err)
					}
					w.WriteHeader(200)
				}

			}

		}
		if r.Method == http.MethodDelete {
			if jsonReq["id"] == nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			} else {
				var id = jsonReq["id"].(string)
				var removed = removeInfo(id)
				if removed {
					w.WriteHeader(200)
				} else if !removed {
					http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				}
			}

		}

	}
	http.HandleFunc("/", server)
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		panic(err)
	}

}

func HttpRequest(method string, url string, headers map[string]interface{}, body map[string]interface{}) *http.Response {

	client := &http.Client{}

	if method == "GET" && len(body) == 0 {
		req, _ := http.NewRequest(method, url, nil)
		if headers != nil {
			for key, value := range headers {

				req.Header.Add(key, value.(string))
			}

		}

		resp, _ := client.Do(req)
		return resp

	} else if method == "POST" {
		out, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, url, bytes.NewBuffer(out))
		if headers != nil {
			for key, value := range headers {
				req.Header.Add(key, value.(string))
			}

		}
		resp, _ := client.Do(req)
		return resp

	} else {
		panic("Метод GET не может иметь тело сообщения. Используйте POST, если вы хотите отправить данные на сервер.")
	}

}
