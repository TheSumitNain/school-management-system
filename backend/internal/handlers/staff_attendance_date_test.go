package handlers

import (
 "testing"
 "net/http"
 "net/http/httptest"
 "github.com/gin-gonic/gin"
)

func TestParseStaffAttendanceDate(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantOK  bool
	}{
		{name: "valid date", value: "2026-11-23", wantOK: true},
		{name: "year one", value: "0001-01-01", wantOK: true},
		{name: "year zero", value: "0000-01-01"},
		{name: "invalid calendar date", value: "2026-02-30"},
		{name: "non canonical date", value: "2026-1-01"},
		{name: "whitespace", value: " 2026-11-23 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseStaffAttendanceDate(tt.value)
			if ok != tt.wantOK {
				t.Fatalf("valid=%v want=%v", ok, tt.wantOK)
			}
		})
	}
}

func TestParseStaffAttendanceMonth(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantOK  bool
	}{
		{name: "valid month", value: "2026-11", wantOK: true},
		{name: "year one", value: "0001-01", wantOK: true},
		{name: "year zero", value: "0000-01"},
		{name: "invalid month", value: "2026-13"},
		{name: "non canonical month", value: "2026-1"},
		{name: "whitespace", value: " 2026-11 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseStaffAttendanceMonth(tt.value)
			if ok != tt.wantOK {
				t.Fatalf("valid=%v want=%v", ok, tt.wantOK)
			}
		})
	}
}

func TestParseStaffAttendancePagination(t *testing.T) {
    tests := []struct { name, limit, offset string; wantLimit, wantOffset int; wantOK bool }{
        {name:"defaults",wantLimit:50,wantOffset:0,wantOK:true},
        {name:"custom",limit:"100",offset:"250",wantLimit:100,wantOffset:250,wantOK:true},
        {name:"zero limit",limit:"0"},
        {name:"oversized limit",limit:"101"},
        {name:"non numeric limit",limit:"many"},
        {name:"negative offset",limit:"50",offset:"-1"},
        {name:"oversized offset",limit:"50",offset:"1000000001"},
        {name:"non numeric offset",limit:"50",offset:"next"},
    }
    for _,tt:=range tests { t.Run(tt.name,func(t *testing.T) {
        router:=gin.New()
        router.GET("/",func(c *gin.Context){limit,offset,ok:=parseStaffAttendancePagination(c);if ok!=tt.wantOK|| (ok&&(limit!=tt.wantLimit||offset!=tt.wantOffset)){c.Status(http.StatusInternalServerError);return};c.Status(http.StatusNoContent)})
        target:="/"
        if tt.limit!="" { target+="?limit="+tt.limit }
        if tt.offset!="" { if tt.limit=="" {target+="?"} else {target+="&"};target+="offset="+tt.offset }
        rec:=httptest.NewRecorder();router.ServeHTTP(rec,httptest.NewRequest(http.MethodGet,target,nil))
        if rec.Code!=http.StatusNoContent {t.Fatalf("status=%d want 204",rec.Code)}
    }) }
}
