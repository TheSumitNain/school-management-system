import { expect, test } from "@playwright/test";

const platformPermissions = [
  "platform.admins.manage",
  "platform.schools.create",
  "platform.school_admins.create",
  "platform.permissions.delegate",
  "platform.schools.read",
];

async function signIn(page: import("@playwright/test").Page, email: string) {
  await page.goto("/");
  await page.getByRole("button", { name: "Platform admin" }).click();
  await page.getByLabel("Email address").fill(email);
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Platform Admin" }).click();
}

test("Platform Admin accounts can be created and safely deactivated with recoverable failures", async ({ page }) => {
  let admins: Array<Record<string, unknown>> = [
    { principal_id: 201, email: "ops@example.com", status: "active", mfa_enrolled: true, created_at: "2026-01-01T00:00:00Z" },
  ];
  let createAttempts = 0;
  let deactivateAttempts = 0;

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/platform-admin/login" && method === "POST") {
      await route.fulfill({ json: { access_token: "e2e-platform-admin-token", token_type: "Bearer", principal: { principal_id: 101, principal_type: "platform_admin", status: "active" } } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: { principal_id: 101, principal_type: "platform_admin", username: "platform-operator", permissions: platformPermissions } });
      return;
    }
    if (path === "/platform/admins" && method === "GET") {
      await route.fulfill({ json: { admins } });
      return;
    }
    if (path === "/platform/admins" && method === "POST") {
      createAttempts += 1;
      const body = request.postDataJSON() as Record<string, unknown>;
      if (createAttempts === 1) {
        await route.fulfill({ status: 409, json: { status: "error", message: "Email already exists.", field_errors: { email: "This email is already used by a Platform Admin." } } });
        return;
      }
      const created = { principal_id: 202, email: String(body.email), status: "active", mfa_enrolled: false, created_at: "2026-11-21T10:00:00Z" };
      admins = [...admins, created];
      await route.fulfill({ status: 201, json: { admin: created, mfa_enrollment_required: true } });
      return;
    }
    if (path === "/platform/admins/201/deactivate" && method === "POST") {
      deactivateAttempts += 1;
      if (deactivateAttempts === 1) {
        await route.fulfill({ status: 503, json: { status: "error", message: "Temporary deactivation failure." } });
        return;
      }
      admins = admins.map(admin => admin.principal_id === 201 ? { ...admin, status: "disabled" } : admin);
      await route.fulfill({ json: { message: "Platform Admin deactivated." } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked platform admin request: ${method} ${path}` } });
  });

  await signIn(page, "platform-operator@example.com");
  await page.getByRole("button", { name: "Create Platform Admin" }).click();
  const createDialog = page.getByRole("dialog", { name: "Create Platform Admin" });
  await createDialog.getByLabel("Email address").fill("security@example.com");
  await createDialog.locator('input[name="password"]').fill("Secure-Initial-Password-2026");
  await createDialog.getByRole("button", { name: "Create Platform Admin" }).click();
  await expect(createDialog.getByText("This email is already used by a Platform Admin.")).toBeVisible();
  await expect(createDialog.getByLabel("Email address")).toHaveValue("security@example.com");
  expect(createAttempts).toBe(1);

  await createDialog.getByRole("button", { name: "Create Platform Admin" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Platform Admin security@example.com created." })).toBeVisible();
  expect(createAttempts).toBe(2);
  await expect(page.getByRole("cell", { name: "security@example.com" })).toBeVisible();

  const targetRow = page.getByRole("row").filter({ hasText: "ops@example.com" });
  await targetRow.getByRole("button", { name: "Deactivate" }).click();
  const deactivateDialog = page.getByRole("dialog", { name: "Deactivate Platform Admin?" });
  await deactivateDialog.getByRole("button", { name: "Deactivate account" }).click();
  await expect(deactivateDialog.getByText("Temporary deactivation failure.")).toBeVisible();
  expect(deactivateAttempts).toBe(1);
  await deactivateDialog.getByRole("button", { name: "Deactivate account" }).click();
  await expect(page.getByRole("status").filter({ hasText: "ops@example.com was deactivated" })).toBeVisible();
  expect(deactivateAttempts).toBe(2);
  await expect(page.getByRole("row").filter({ hasText: "ops@example.com" }).getByText("disabled")).toBeVisible();
});

test("school onboarding refreshes the school list and delegated permissions can be granted and revoked", async ({ page }) => {
  let schools: Array<Record<string, unknown>> = [
    { id: 1, name: "Sunrise Public School", code: "SUNRISE", status: "active", created_at: "2026-01-01T00:00:00Z" },
  ];
  let schoolListCalls = 0;
  let onboardAttempts = 0;
  let delegation: Record<string, unknown> | null = null;
  let grantCalls = 0;
  let revokeCalls = 0;

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/platform-admin/login" && method === "POST") {
      await route.fulfill({ json: { access_token: "e2e-platform-school-token", token_type: "Bearer", principal: { principal_id: 102, principal_type: "platform_admin", status: "active" } } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: { principal_id: 102, principal_type: "platform_admin", username: "platform-onboarder", permissions: platformPermissions } });
      return;
    }
    if (path === "/platform/admins" && method === "GET") { await route.fulfill({ json: { admins: [] } }); return; }
    if (path === "/platform/schools" && method === "GET") {
      schoolListCalls += 1;
      await route.fulfill({ json: { schools } });
      return;
    }
    if (path === "/platform/permissions/delegable") {
      await route.fulfill({ json: { permissions: [{ permission_key: "students.read_basic", description: "View basic student information.", is_delegable: true }] } });
      return;
    }
    if (path === "/platform/schools" && method === "POST") {
      onboardAttempts += 1;
      if (onboardAttempts === 1) {
        await route.fulfill({ status: 409, json: { status: "error", message: "School code already exists.", field_errors: { code: "Choose a unique school code." } } });
        return;
      }
      const created = { id: 2, name: "Sunrise Academy", code: "SUNRISE02", status: "active", created_at: "2026-11-21T10:00:00Z" };
      schools = [...schools, created];
      await route.fulfill({ status: 201, json: { school: created, school_admin: { principal_id: 220, username: "sunriseadmin" } } });
      return;
    }
    if (path === "/platform/schools/1/permission-delegations" && method === "GET") {
      await route.fulfill({ json: { delegations: [] } });
      return;
    }
    if (path === "/platform/schools/2/permission-delegations" && method === "GET") {
      await route.fulfill({ json: { delegations: delegation ? [delegation] : [] } });
      return;
    }
    if (path === "/platform/schools/2/permission-delegations" && method === "POST") {
      grantCalls += 1;
      const body = request.postDataJSON() as Record<string, unknown>;
      delegation = { id: 301, school_id: 2, permission_key: String(body.permission_key), can_assign_to_roles: Boolean(body.can_assign_to_roles), can_override_users: Boolean(body.can_override_users), granted_by: 102, created_at: "2026-11-21T10:00:00Z" };
      await route.fulfill({ status: 201, json: { delegation, message: "Permission delegated." } });
      return;
    }
    if (path === "/platform/permission-delegations/301/revoke" && method === "POST") {
      revokeCalls += 1;
      delegation = null;
      await route.fulfill({ json: { message: "Permission revoked." } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked platform school request: ${method} ${path}` } });
  });

  await signIn(page, "platform-onboarder@example.com");
  await page.getByRole("tab", { name: "Onboard school" }).click();
  await page.getByRole("button", { name: "Onboard a new school" }).click();
  const schoolDialog = page.getByRole("dialog", { name: "Onboard a new school" });
  await schoolDialog.getByLabel("School name").fill("Sunrise Academy");
  await schoolDialog.getByLabel("School code").fill("SUNRISE02");
  await schoolDialog.getByLabel("School Admin username").fill("sunriseadmin");
  await schoolDialog.locator('input[name="school_admin_initial_password"]').fill("Secure-School-Admin-2026");
  await schoolDialog.getByRole("button", { name: "Create school and School Admin" }).click();
  await expect(schoolDialog.getByText("Choose a unique school code.")).toBeVisible();
  await expect(schoolDialog.getByLabel("School name")).toHaveValue("Sunrise Academy");
  expect(onboardAttempts).toBe(1);

  await schoolDialog.getByRole("button", { name: "Create school and School Admin" }).click();
  await expect(page.getByRole("status").filter({ hasText: "School Sunrise Academy (SUNRISE02) created." })).toBeVisible();
  expect(onboardAttempts).toBe(2);

  await page.getByRole("tab", { name: "School permissions" }).click();
  await expect.poll(() => schoolListCalls).toBeGreaterThan(1);
  const schoolSelect = page.getByLabel("School");
  await expect(schoolSelect.locator("option", { hasText: "Sunrise Academy" })).toHaveCount(1);
  await schoolSelect.selectOption("2");
  await page.getByLabel("Permission").selectOption("students.read_basic");
  await page.getByRole("button", { name: "Grant permission to school" }).click();
  await expect(page.getByText("students.read_basic", { exact: true })).toBeVisible();
  expect(grantCalls).toBe(1);

  await page.getByRole("button", { name: "Revoke" }).click();
  const revokeDialog = page.getByRole("dialog", { name: "Revoke school permission?" });
  await revokeDialog.getByRole("button", { name: "Cancel" }).click();
  await expect(revokeDialog).not.toBeVisible();
  expect(revokeCalls).toBe(0);

  await page.getByRole("button", { name: "Revoke" }).click();
  const confirmDialog = page.getByRole("dialog", { name: "Revoke school permission?" });
  await confirmDialog.getByRole("button", { name: "Revoke permission" }).click();
  await expect(page.getByText("No active permission delegations for this school.")).toBeVisible();
  expect(revokeCalls).toBe(1);
});
