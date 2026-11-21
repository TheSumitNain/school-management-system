-- V57: add school-scoped staff qualifications, certifications and experience.
-- Records are retained as staff history and are never cascaded across tenants.

ALTER TABLE public.staff_profiles
    ADD CONSTRAINT uq_staff_profiles_principal_school
    UNIQUE (principal_id, school_id);

CREATE TABLE public.staff_qualifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    staff_principal_id BIGINT NOT NULL,
    qualification_type VARCHAR(30) NOT NULL
        CHECK (qualification_type IN ('school', 'diploma', 'undergraduate', 'postgraduate', 'doctorate', 'other')),
    qualification_name VARCHAR(160) NOT NULL
        CHECK (length(btrim(qualification_name)) BETWEEN 1 AND 160),
    specialization VARCHAR(160),
    institution VARCHAR(200) NOT NULL
        CHECK (length(btrim(institution)) BETWEEN 1 AND 200),
    board_university VARCHAR(200),
    completion_year SMALLINT
        CHECK (completion_year IS NULL OR completion_year BETWEEN 1900 AND 2200),
    score NUMERIC(8,3)
        CHECK (score IS NULL OR score >= 0),
    score_scale VARCHAR(20)
        CHECK (score_scale IS NULL OR score_scale IN ('percentage', 'cgpa', 'gpa', 'marks', 'grade')),
    credential_reference VARCHAR(120),
    verification_status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (verification_status IN ('pending', 'verified', 'rejected')),
    verification_notes VARCHAR(1000),
    verified_by_principal_id BIGINT,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT staff_qualifications_staff_school_fkey
        FOREIGN KEY (staff_principal_id, school_id)
        REFERENCES public.staff_profiles (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT staff_qualifications_verifier_school_fkey
        FOREIGN KEY (verified_by_principal_id, school_id)
        REFERENCES public.users (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT staff_qualifications_score_pair_check
        CHECK ((score IS NULL AND score_scale IS NULL) OR (score IS NOT NULL AND score_scale IS NOT NULL)),
    CONSTRAINT staff_qualifications_verification_actor_check
        CHECK (
            (verification_status = 'pending' AND verified_by_principal_id IS NULL AND verified_at IS NULL)
            OR
            (verification_status IN ('verified', 'rejected')
                AND verified_by_principal_id IS NOT NULL AND verified_at IS NOT NULL)
        )
);

CREATE TABLE public.staff_certifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    staff_principal_id BIGINT NOT NULL,
    certificate_name VARCHAR(180) NOT NULL
        CHECK (length(btrim(certificate_name)) BETWEEN 1 AND 180),
    issuing_organization VARCHAR(200) NOT NULL
        CHECK (length(btrim(issuing_organization)) BETWEEN 1 AND 200),
    issue_date DATE,
    expiry_date DATE,
    credential_id VARCHAR(120),
    verification_status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (verification_status IN ('pending', 'verified', 'rejected')),
    verification_notes VARCHAR(1000),
    verified_by_principal_id BIGINT,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT staff_certifications_staff_school_fkey
        FOREIGN KEY (staff_principal_id, school_id)
        REFERENCES public.staff_profiles (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT staff_certifications_verifier_school_fkey
        FOREIGN KEY (verified_by_principal_id, school_id)
        REFERENCES public.users (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT staff_certifications_date_order_check
        CHECK (issue_date IS NULL OR expiry_date IS NULL OR expiry_date >= issue_date),
    CONSTRAINT staff_certifications_verification_actor_check
        CHECK (
            (verification_status = 'pending' AND verified_by_principal_id IS NULL AND verified_at IS NULL)
            OR
            (verification_status IN ('verified', 'rejected')
                AND verified_by_principal_id IS NOT NULL AND verified_at IS NOT NULL)
        )
);

CREATE TABLE public.staff_experience (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    staff_principal_id BIGINT NOT NULL,
    employer_name VARCHAR(200) NOT NULL
        CHECK (length(btrim(employer_name)) BETWEEN 1 AND 200),
    designation VARCHAR(120) NOT NULL
        CHECK (length(btrim(designation)) BETWEEN 1 AND 120),
    employment_type VARCHAR(20) NOT NULL
        CHECK (employment_type IN ('permanent', 'contract', 'part_time', 'temporary', 'internship', 'other')),
    start_date DATE NOT NULL,
    end_date DATE,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    description VARCHAR(2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT staff_experience_staff_school_fkey
        FOREIGN KEY (staff_principal_id, school_id)
        REFERENCES public.staff_profiles (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT staff_experience_date_order_check
        CHECK (end_date IS NULL OR end_date >= start_date),
    CONSTRAINT staff_experience_current_end_date_check
        CHECK ((is_current AND end_date IS NULL) OR (NOT is_current))
);

CREATE INDEX ix_staff_qualifications_school_staff
    ON public.staff_qualifications (school_id, staff_principal_id, created_at DESC, id DESC);
CREATE INDEX ix_staff_certifications_school_staff
    ON public.staff_certifications (school_id, staff_principal_id, created_at DESC, id DESC);
CREATE INDEX ix_staff_experience_school_staff
    ON public.staff_experience (school_id, staff_principal_id, start_date DESC, id DESC);

COMMENT ON TABLE public.staff_qualifications IS
    'School-scoped education records for a staff employee; mutations are audited by the application.';
COMMENT ON TABLE public.staff_certifications IS
    'School-scoped professional certification records for a staff employee; mutations are audited by the application.';
COMMENT ON TABLE public.staff_experience IS
    'School-scoped prior/current employment history for a staff employee; mutations are audited by the application.';
