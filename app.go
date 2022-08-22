package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
)

const dbName = "requests.db"

type Response struct {
	Headers map[string]any
	Length  int
	Status  int
}
type Request struct {
	Headers map[string]string `json:"headers"`
	Body    map[string]string `json:"body"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
}

type MainReq struct {
	Id       string
	Response Response
	Request  Request
}

type ReqId struct {
	Id string
}

func addInfo(data MainReq, sqlInstance *sql.DB) error {
	/*
		Добавляет запрос в БД
	*/

	var ErrorAddInfo error
	var headersResp, reqErr = json.Marshal(data.Response.Headers)
	if reqErr != nil {
		ErrorAddInfo = reqErr
	}
	var headersReq, respErr = json.Marshal(data.Request.Headers)
	if respErr != nil {
		ErrorAddInfo = respErr
	}

	var body, bodyErr = json.Marshal(data.Request.Body)
	if bodyErr != nil {
		ErrorAddInfo = bodyErr
	}

	records := `INSERT INTO req_and_response(IdReq, HeadersResp, Length, Status, HeadersReq, Body, Method, Url) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	query, prepareError := sqlInstance.Prepare(records)
	if prepareError != nil {
		ErrorAddInfo = prepareError
	}

	_, execError := query.Exec(data.Id, headersResp, data.Response.Length, data.Response.Status, headersReq, body, data.Request.Method, data.Request.Url)
	if execError != nil {
		ErrorAddInfo = execError
	}
	return ErrorAddInfo
}

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

func createTable(sqlInstance *sql.DB) {
	/*
		Создает таблицу req_and_response
	*/

	users_table := `CREATE TABLE req_and_response (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "IdReq" TEXT,
        "HeadersResp" BLOB,
        "Length" INT,
        "Status" INT,
        "HeadersReq" BLOB,
        "Body" BLOB,
        "Method" TEXT,
        "Url" TEXT);`
	query, prepareError := sqlInstance.Prepare(users_table)
	if prepareError != nil {
		log.Fatal(prepareError)
	}
	_, execError := query.Exec()

	if execError != nil {
		log.Fatal(execError)
	}

	fmt.Println("Table created successfully!")
}

func fetchRequests(sqlInstance *sql.DB) []map[string]any {
	/*
		Выкачивает всю инфу из БД
	*/

	var results []map[string]any
	record, queryError := sqlInstance.Query("SELECT * FROM req_and_response")

	if queryError != nil {
		log.Fatal(queryError)
	}

	defer func(record *sql.Rows) {
		err := record.Close()
		if err != nil {

		}
	}(record)

	for record.Next() {
		var res = map[string]any{}
		var id int
		var IdReq string
		var HeadersResp []byte
		var Length int
		var Status int
		var HeadersReq []byte
		var Body []byte
		var Method string
		var Url string
		scanError := record.Scan(&id, &IdReq, &HeadersResp, &Length, &Status, &HeadersReq, &Body, &Method, &Url)

		if scanError != nil {
			log.Fatal(scanError)
		}

		//var res = map[string]any{id, IdReq, HeadersResp, Length, Status, HeadersReq, Body, Method, Url}
		res["id"] = id
		res["IdReq"] = IdReq
		res["HeadersResp"] = HeadersResp
		res["Length"] = Length
		res["Status"] = Status
		res["HeadersReq"] = HeadersReq
		res["Body"] = Body
		res["Method"] = Method
		res["Url"] = Url
		results = append(results, res)
	}

	return results

}

func removeInfo(id string, sqlInstance *sql.DB) bool {
	/*
		Удаляет запрос из Бд по id
	*/

	var idDb = IdFromDb(id, sqlInstance)
	if idDb == -1 {
		return false
	} else {

		var deleteReq = fmt.Sprintf("delete from req_and_response where id = %d", idDb)
		fmt.Println("deleteReq: ", deleteReq)
		_, execError := sqlInstance.Exec(deleteReq)

		if execError != nil {
			log.Fatal(execError)
		}

		return true

	}

}

func GetMD5Hash(text string) string {
	/*
		Генерирует хэш из строки
	*/

	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func jsonResp(data map[string]any) []byte {
	/*
		Генерирует Json ответ
	*/

	var headersReq = map[string]string{}
	var unmarshalError = json.Unmarshal(data["HeadersReq"].([]byte), &headersReq)

	if unmarshalError != nil {
		log.Fatal(unmarshalError)
	}

	var headersResp = map[string]any{}
	var unmarshalError2 = json.Unmarshal(data["HeadersResp"].([]byte), &headersResp)

	if unmarshalError2 != nil {
		log.Fatal(unmarshalError2)
	}

	var body = map[string]string{}
	var unmarshalError3 = json.Unmarshal(data["Body"].([]byte), &body)

	if unmarshalError3 != nil {
		log.Fatal(unmarshalError3)
	}

	var req = Request{Headers: headersReq,
		Body:   body,
		Method: (data["Method"]).(string), Url: (data["Url"]).(string)}

	var resp = Response{Headers: headersResp,
		Length: (data["Length"]).(int),
		Status: (data["Status"]).(int)}
	var result = map[string]any{"id": data["IdReq"], "request": req, "response": resp}
	var jsonResult, jsonError = json.MarshalIndent(result, "", "    ")

	if jsonError != nil {
		log.Fatal(jsonError)
	}

	return jsonResult

}

func searchById(id string, sqlInstance *sql.DB) []byte {
	/*
		Делает поиск внутри БД по id, если id совпадают, возвращает сохраненный с ним request+response
	*/

	var requests = fetchRequests(sqlInstance)

	for _, requestIter := range requests {
		if requestIter["IdReq"] == id {
			return jsonResp(requestIter)
		}

	}
	return nil

}

func IdFromDb(id string, sqlInstance *sql.DB) int {
	/*
		Делает поиск внутри БД по id, если id совпадают, возвращает id строки из БД
	*/

	var requests = fetchRequests(sqlInstance)

	for _, requestIter := range requests {
		if requestIter["IdReq"] == id {
			return requestIter["id"].(int)
		}

	}

	return -1

}

func CacheLRU(request Request, sqlInstance *sql.DB) ([]byte, error) {
	/*
		Проверяет, существует ли подобный request в БД, если да, возвращает request+response
	*/
	var ErrorCacheLRU error
	var headers = fmt.Sprintf("%s", request.Headers)
	var body = fmt.Sprintf("%s", request.Body)
	var requests = fetchRequests(sqlInstance)
	for _, requestIter := range requests {
		var methodLocal = (requestIter["Method"]).(string)
		var urlLocal = (requestIter["Url"]).(string)

		var headerstLocal = map[string]string{}
		var _ = json.Unmarshal(requestIter["HeadersReq"].([]byte), &headerstLocal)

		var bodyLocal = map[string]string{}
		var unmarshalError = json.Unmarshal(requestIter["Body"].([]byte), &bodyLocal)

		if unmarshalError != nil {
			ErrorCacheLRU = unmarshalError
		}

		var headerstLocalString = fmt.Sprintf("%s", headerstLocal)
		var bodyLocalString = fmt.Sprintf("%s", bodyLocal)
		if methodLocal == request.Method {
			if urlLocal == request.Url {
				if GetMD5Hash(headerstLocalString) == GetMD5Hash(headers) {
					if GetMD5Hash(bodyLocalString) == GetMD5Hash(body) {
						return jsonResp(requestIter), ErrorCacheLRU
					} else if len(body) == 0 && bodyLocal == nil {
						return jsonResp(requestIter), ErrorCacheLRU
					}

				} else if len(headers) == 0 && headerstLocal == nil {
					if GetMD5Hash(bodyLocalString) == GetMD5Hash(body) {
						return jsonResp(requestIter), ErrorCacheLRU
					} else if len(body) == 0 && bodyLocal == nil {
						return jsonResp(requestIter), ErrorCacheLRU
					}

				}

			}

		}

	}

	return nil, ErrorCacheLRU

}

func methodPost(decoder []byte, responseWriter http.ResponseWriter, sqlInstance *sql.DB) error {

	var jsonReq = Request{}
	var unmarshalError = json.Unmarshal(decoder, &jsonReq)

	if unmarshalError != nil {
		return unmarshalError
	}

	var body = map[string]string{}
	var headers = map[string]string{}
	if jsonReq.Body == nil {
		body = nil
	} else {
		body = jsonReq.Body
	}

	if jsonReq.Headers == nil {
		headers = nil

	} else {
		headers = jsonReq.Headers
	}
	var method = jsonReq.Method
	var url = jsonReq.Url

	var jsR = Request{Method: jsonReq.Method, Url: url, Headers: jsonReq.Headers, Body: jsonReq.Body}
	var cacheLRU, ErrorCacheLRU = CacheLRU(jsR, sqlInstance)
	if ErrorCacheLRU != nil {
		return ErrorCacheLRU
	}
	if cacheLRU == nil {
		var uuidForReq = uuid()
		var httpResponse, ErrorHttpRequest = HttpRequest(method, url, headers, body)
		if ErrorHttpRequest != nil {
			return ErrorHttpRequest
		}
		if httpResponse.StatusCode != 400 {
			var contentType = httpResponse.Header["Content-Type"][0]
			var secondHeaders = map[string]any{"Content-Length": httpResponse.ContentLength, "Content-Type": contentType}
			var response = map[string]any{"status": httpResponse.StatusCode, "headers": secondHeaders, "length": httpResponse.ContentLength, "id": uuidForReq}
			var headersData = MainReq{Id: uuidForReq, Request: Request{Headers: headers, Body: body, Method: method, Url: url},
				Response: Response{Headers: secondHeaders, Length: int(httpResponse.ContentLength), Status: httpResponse.StatusCode}}
			ErrorAddInfo := addInfo(headersData, sqlInstance)
			if ErrorAddInfo != nil {
				return ErrorAddInfo
			}

			var dataToWatch, jsonError = json.MarshalIndent(response, "", "    ")

			if jsonError != nil {
				return jsonError
			}

			_, writeError := responseWriter.Write(dataToWatch)
			if writeError != nil {
				return writeError
			}

			responseWriter.WriteHeader(200)
		} else {

			responseWriter.WriteHeader(400)
			http.Error(responseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}

	} else {
		_, writeError2 := responseWriter.Write(cacheLRU)
		if writeError2 != nil {
			return writeError2
		}

		responseWriter.WriteHeader(200)
	}
	return nil

}

func methodGet(decoder []byte, responseWriter http.ResponseWriter, sqlInstance *sql.DB) error {
	/*
		Метод возвращает request по id
	*/

	var jsonReq = ReqId{}
	unmarshalError := json.Unmarshal(decoder, &jsonReq)
	if unmarshalError != nil {
		return unmarshalError
	}

	if jsonReq.Id == "" {
		http.Error(responseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	} else {
		var text = searchById(jsonReq.Id, sqlInstance)
		if text == nil {
			http.Error(responseWriter, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			_, writeError := responseWriter.Write(text)
			if writeError != nil {
				return writeError
			}

			responseWriter.WriteHeader(200)
		}
	}
	return nil

}

func methodDelete(decoder []byte, responseWriter http.ResponseWriter, sqlInstance *sql.DB) error {
	/*
		Метод удаляет request по id
	*/

	var jsonReq = ReqId{}
	unmarshalError := json.Unmarshal(decoder, &jsonReq)
	if unmarshalError != nil {
		return unmarshalError
	}

	if jsonReq.Id == "" {
		http.Error(responseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	} else {
		var id = jsonReq.Id
		var removed = removeInfo(id, sqlInstance)
		if removed {
			http.Error(responseWriter, http.StatusText(http.StatusOK), http.StatusOK)
			responseWriter.WriteHeader(200)
		} else if !removed {
			http.Error(responseWriter, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}

	}

	return nil

}
func main() {
	var server = func(w http.ResponseWriter, r *http.Request) {
		db, dateBaseError := sql.Open("sqlite3", dbName)
		if dateBaseError != nil {
			panic(dateBaseError)
		}
		_, errCheckDb := db.Query("SELECT * FROM req_and_response")
		if errCheckDb != nil {
			createTable(db)
		}

		decoder, errReadAll := ioutil.ReadAll(r.Body)
		if errReadAll != nil {
			panic(errReadAll)
		}

		if r.Method == http.MethodPost {
			ErrorMethodPost := methodPost(decoder, w, db)
			if ErrorMethodPost != nil {
				panic(ErrorMethodPost)
			}
		}

		if r.Method == http.MethodGet {
			ErrorMethodGet := methodGet(decoder, w, db)
			if ErrorMethodGet != nil {
				panic(ErrorMethodGet)
			}
		}

		if r.Method == http.MethodDelete {
			ErrorMethodDelete := methodDelete(decoder, w, db)
			if ErrorMethodDelete != nil {
				panic(ErrorMethodDelete)
			}
		}

	}

	http.HandleFunc("/", server)
	listenError := http.ListenAndServe(":8000", nil)
	if listenError != nil {
		log.Fatal(listenError)
	}

}

func HttpRequest(method string, url string, headers map[string]string, body map[string]string) (*http.Response, error) {
	/*
		Отправляет запрос по указанному url
	*/
	var ErrorHttpRequest error
	client := &http.Client{}
	if method == "GET" && len(body) == 0 {
		req, reqError := http.NewRequest(method, url, nil)

		if reqError != nil {
			ErrorHttpRequest = reqError
		}

		if headers != nil {
			for key, value := range headers {
				req.Header.Add(key, value)
			}
		}
		resp, doError := client.Do(req)

		if doError != nil {
			ErrorHttpRequest = doError
		}

		return resp, ErrorHttpRequest

	} else if method == "POST" {
		out, jsonError := json.Marshal(body)

		if jsonError != nil {
			ErrorHttpRequest = jsonError
		}

		req, reqError := http.NewRequest(method, url, bytes.NewBuffer(out))

		if reqError != nil {
			ErrorHttpRequest = reqError
		}

		if headers != nil {
			for key, value := range headers {
				req.Header.Add(key, value)
			}
		}
		resp, doError2 := client.Do(req)

		if doError2 != nil {
			ErrorHttpRequest = doError2
		}

		return resp, ErrorHttpRequest

	} else if method != "" && url != "" {
		resp := &http.Response{Status: "400 Bad Request", StatusCode: 400}
		return resp, nil
	} else {
		resp := &http.Response{Status: "400 Bad Request", StatusCode: 400}
		return resp, nil
	}

}
