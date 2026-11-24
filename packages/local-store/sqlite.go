package localstore

import("database/sql";"net/url";"path/filepath";"strings";_ "modernc.org/sqlite")
// Open configures each physical connection, including replacement connections after an error.
func Open(path string)(*sql.DB,error){
 absolute,err:=filepath.Abs(path);if err!=nil{return nil,err}
 filePath:=filepath.ToSlash(absolute);if !strings.HasPrefix(filePath,"/"){filePath="/"+filePath}
 uri:=url.URL{Scheme:"file",Path:filePath};q:=uri.Query()
 q.Add("_pragma","foreign_keys(1)");q.Add("_pragma","synchronous(FULL)");q.Add("_pragma","busy_timeout(5000)");q.Set("_txlock","immediate");uri.RawQuery=q.Encode()
 db,err:=sql.Open("sqlite",uri.String());if err==nil{db.SetMaxOpenConns(1)};return db,err
}
