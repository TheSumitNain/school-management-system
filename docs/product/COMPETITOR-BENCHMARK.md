# School ERP Competitor Benchmark

**Reviewed:** 2026-11-27  
**Purpose:** Compare publicly documented workflows from CoreAxis, Fedena and Entab CampusCare to identify practical usability patterns for SchoolDesk. This is a product benchmark, not a claim that every advertised capability has been independently tested.

## Executive findings

The strongest recurring pattern across established school platforms is **role-specific workspaces**: a teacher should land on teaching tasks, an accountant on finance tasks, and school leadership on school-wide operations. A dashboard should explain what needs attention and provide direct paths into the relevant workflow—not merely display a few totals.

## Public feature comparison

| Area | CoreAxis School ERP | Fedena | Entab CampusCare |
|---|---|---|---|
| Role-based access | Separate Admin/Principal, Teacher, Accountant and Support Staff experiences; menus depend on role. | Dashboards/logins for teachers, non-teaching staff, students, parents and management; privileges control special tasks. | Separate mobile experiences for parents, teachers, principals and admins. |
| Teacher workflow | “Today” page with class-attendance status, timetable, homework, marks, notices and personal leave/attendance. | Teacher login includes timetable, marks entry, assignments, attendance and leave. | Teacher app includes attendance, homework/assignments, marks, lesson plans, leave and payslips. |
| Staff/HR | Staff profiles, staff attendance, leave, payroll/payslips and substitutions. Support roles can be recorded without a login. | HR, employee management, attendance, leave, payroll and role-based privileges. | Staff attendance, payroll, leave approvals and CPD tracking are advertised. |
| Student/parent | Parent/student portal for attendance, fees, results, homework and notices; one parent login can cover multiple children. | Student/parent login for attendance, grades, timetable, assignments, fees and communication. | Parent app covers attendance alerts/history, homework, results, fees, calendar, transport and teacher communication. |
| Academics | Classes, subjects, timetable, homework, attendance, exams, marks, report cards and online exams. | Courses/batches, timetable, attendance, gradebook, examinations, assignments and online learning integrations. | Admissions, assessments, report cards, homework, online classes, academic calendar and syllabus. |
| Finance | Fee structures, receipts, online payments, dues, expenses, finance ledger and payroll. | Finance, fee collection, payment gateway integrations and finance reports. | Fee structures/installments, online payments, receipts, dues and financial reports. |
| Campus operations | Library, transport, hostel, inventory, holidays and school store. | Library, transport, hostel, inventory and events/calendar. | Library, transport/GPS, visitor management, student welfare and school diary features. |
| Communication | Notices, SMS/email, gallery, competitions, letters and public school website. | Messaging, news, SMS/email, calendar, discussion/task plugins and mobile notifications. | Circulars, announcements, chat, SMS/email/WhatsApp/app notifications and diary remarks. |
| Reports/analytics | Clickable, class/date-filtered charts with drill-down to source records; CSV reports. | Report Center, audit and data export modules; multiple dashboards. | Management dashboards, school statistics, login history and analytics are advertised. |
| Documents | ID cards, certificates and TCs with QR verification; document register. | Certificate/ID card generators; document manager in higher module tiers. | Student records, document upload/verification and compliance-oriented reporting. |

## What SchoolDesk should adopt

### P0 — Fix usability and access first

1. **Correct default role provisioning:** when a teaching employee login is created, assign the existing school-scoped Teacher role transactionally and audit it. Do not give teaching permissions to non-teaching staff by default.
2. **Role-aware landing pages:** Admin/Principal, Teacher and approved staff should see the work relevant to their role. For a teacher, the natural starting point is assigned classes, today's timetable, attendance status and pending work—but only when the required APIs and assignment scopes exist.
3. **Visible access explanation:** show the user's primary/additional roles and effective permissions in a readable way. If no work area is assigned, explain exactly whom to contact and why the workspace is limited.
4. **Action-first navigation:** provide direct shortcuts to the next permitted task, while retaining the full permission-filtered navigation.
5. **Useful empty states:** explain whether data is missing, setup is incomplete, or access is not granted. Never replace missing data with fabricated demo counts.

### P1 — Complete existing school workflows

- Staff self-service for own attendance, leave and relevant employment information.
- School-wide operational reports that respect role and school scope.
- Teacher assignment-aware timetable and attendance workflows.
- Consistent notifications and action-required indicators.

### Later phases — Only after core web workflows are stable

- Parent/student portals and mobile apps.
- Fees, payroll and finance workflows.
- Library, transport, hostel, inventory and advanced campus operations.
- Document generation/verification and advanced analytics.

These are benchmark opportunities, not automatic scope additions. Prioritize only after the current SchoolDesk workflows are usable, secure and backed by real APIs.

## Important security notes

- A role's permission is not a substitute for resource-level authorization. Teacher access must still be restricted to explicitly assigned classes/subjects/students by the backend.
- Do not grant school-wide attendance, leave or student access to non-teaching employees merely to make the dashboard look populated.
- Support/operational employees who do not need a login should remain record-only until a separate, approved access model exists.
- Keep tenant isolation, explicit DENY overrides, audit history and server-side authorization authoritative.

## Public sources reviewed

### CoreAxis School ERP
- Features: https://coreaxis.info/features
- User manual and role guides: https://coreaxis.info/manual/
- Dashboard: https://coreaxis.info/manual/dashboard.html
- Teacher guide: https://coreaxis.info/manual/guide-teacher.html
- Staff guide: https://coreaxis.info/manual/staff.html
- Analytics: https://coreaxis.info/manual/analytics.html
- Parent/student portal: https://coreaxis.info/manual/portal.html

### Fedena
- Feature tour: https://fedena.com/feature-tour
- Employee/Teacher login: https://fedena.com/feature-tour/employee-teacher-login
- Mobile app: https://fedena.com/mobile_app
- Support process guide: https://support.fedena.com/support/solutions/articles/266503-process-flow-start-using-fedena

### Entab CampusCare
- CampusCare ERP: https://www.campuscare.in/
- CampusCare mobile apps: https://www.entab.in/mobile-apps.html
- CampusCare 10X ERP: https://www.entab.in/school-erp-software.html
- CampusCare 10X mobile app: https://www.entab.in/mobile-application.html

*Vendor websites/manuals describe advertised capabilities. Before copying a workflow, verify it in a live demo, confirm actual API/security behavior, and assess fit with SchoolDesk's approved scope.*
