package repository

import (
 "errors"
 "testing"
)

func TestValidateStaffLeaveRequestDateFilters(t *testing.T) {
 tests:=[]struct{name,onDate,fromDate,toDate string;wantErr bool}{
  {name:"no filters"},
  {name:"single date",onDate:"2026-11-23"},
  {name:"valid range",fromDate:"2026-11-01",toDate:"2026-11-30"},
  {name:"date and range",onDate:"2026-11-12",fromDate:"2026-11-01",toDate:"2026-11-30"},
  {name:"invalid on date",onDate:"23-11-2026",wantErr:true},
  {name:"nonexistent calendar date",onDate:"2026-02-30",wantErr:true},
  {name:"year zero is not a PostgreSQL date",onDate:"0000-01-01",wantErr:true},
  {name:"year zero in date range",fromDate:"0000-01-01",toDate:"0001-01-01",wantErr:true},
  {name:"range missing end",fromDate:"2026-11-01",wantErr:true},
  {name:"range missing start",toDate:"2026-11-30",wantErr:true},
  {name:"invalid range start",fromDate:"not-a-date",toDate:"2026-11-30",wantErr:true},
  {name:"invalid range end",fromDate:"2026-11-01",toDate:"2026-02-30",wantErr:true},
  {name:"reversed range",fromDate:"2026-11-30",toDate:"2026-11-01",wantErr:true},
 }
 for _,tt:=range tests { t.Run(tt.name,func(t *testing.T){
  err:=validateStaffLeaveRequestDateFilters(tt.onDate,tt.fromDate,tt.toDate)
  if tt.wantErr && !errors.Is(err,ErrInvalidStaffLeave) { t.Fatalf("expected ErrInvalidStaffLeave, got %v",err) }
  if !tt.wantErr && err!=nil { t.Fatalf("unexpected error: %v",err) }
 }) }
}

func TestParseStaffLeaveDate(t *testing.T) {
 tests:=[]struct{name,value string;wantErr bool}{
  {name:"valid date",value:"2026-11-23"},
  {name:"year one",value:"0001-01-01"},
  {name:"year zero",value:"0000-01-01",wantErr:true},
  {name:"invalid calendar date",value:"2026-02-30",wantErr:true},
  {name:"non canonical date",value:"2026-1-01",wantErr:true},
 }
 for _,tt:=range tests { t.Run(tt.name,func(t *testing.T){
  _,err:=parseStaffLeaveDate(tt.value)
  if tt.wantErr && !errors.Is(err,ErrInvalidStaffLeave) { t.Fatalf("expected ErrInvalidStaffLeave, got %v",err) }
  if !tt.wantErr && err!=nil { t.Fatalf("unexpected error: %v",err) }
 }) }
}
