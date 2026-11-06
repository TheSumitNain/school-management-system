package repository

import (
	"context"
	"testing"
)

// TestGuardianPermissionDelegationV19Integration verifies the migration makes
// guardian permissions explicitly delegable while retaining V4's runtime
// protection of permission identity and delegability.
func TestGuardianPermissionDelegationV19Integration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		SELECT permission_key, is_delegable
		FROM permissions
		WHERE scope_type='school'
		  AND permission_key IN ('guardians.read','guardians.manage')
		ORDER BY permission_key
	`)
	if err != nil { t.Fatalf("query guardian permission catalogue: %v", err) }
	defer rows.Close()
	found:=map[string]bool{}
	for rows.Next(){
		var key string
		var delegable bool
		if err:=rows.Scan(&key,&delegable);err!=nil{t.Fatalf("scan guardian permission: %v",err)}
		found[key]=delegable
	}
	if err:=rows.Err();err!=nil{t.Fatalf("iterate guardian permissions: %v",err)}
	if value,ok:=found["guardians.read"];!ok||!value{t.Fatalf("guardians.read must be present and delegable; got %v",found)}
	if value,ok:=found["guardians.manage"];!ok||!value{t.Fatalf("guardians.manage must be present and delegable; got %v",found)}

	for _,key:=range []string{"guardians.read","guardians.manage"}{
		tx,err:=pool.Begin(ctx)
		if err!=nil{t.Fatalf("begin permission guard test: %v",err)}
		_,err=tx.Exec(ctx,`UPDATE permissions SET is_delegable=FALSE WHERE permission_key=$1`,key)
		_ = tx.Rollback(ctx)
		if err==nil{t.Fatalf("V4 catalogue trigger must reject runtime delegability changes for %s",key)}
	}
}
