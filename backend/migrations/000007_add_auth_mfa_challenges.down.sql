DROP TRIGGER IF EXISTS trg_validate_auth_mfa_challenge_principal ON auth_mfa_challenges;
DROP FUNCTION IF EXISTS validate_auth_mfa_challenge_principal();
DROP TABLE IF EXISTS auth_mfa_challenges;
