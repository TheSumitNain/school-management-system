-- V26 rollback: remove timetable schedules before their period definitions.
DROP TRIGGER IF EXISTS trg_guard_v26_teacher_assignment_schedule ON public.teacher_subject_assignments;
DROP FUNCTION IF EXISTS public.guard_v26_teacher_assignment_schedule();
DROP FUNCTION IF EXISTS public.validate_v26_timetable_entry();
DROP TABLE IF EXISTS timetable_entries;
DROP TABLE IF EXISTS timetable_periods;
ALTER TABLE teacher_subject_assignments
    DROP CONSTRAINT IF EXISTS teacher_subject_assignments_timetable_scope_unique;
