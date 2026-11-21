import { expect, test } from "@playwright/test";

test("custom roles can be created, granted delegated permissions and safely revoked", async ({ page }) => {
  let roles: Array<Record<string, unknown>> = [
    { id: 2, role_key: "school_admin", name: "School Admin", is_system: true, created_at: "2026-01-01T00:00:00Z" },
    { id: 9, role_key: "library_coordinator", name: "Library Coordinator", is_system: false, created_at: "2026-01-01T00:00:00Z" },
  ];
  let grants: Array<Record<string, unknown>> = [];
  let createCalls = 0;
  let grantCalls = 0;
  let revokeCalls = 0;
  let failCreate = true;
  let failGrant = true;
  let failRevoke = true;

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/login" && method === "POST") {
      await route.fulfill({ json: { access_token: "e2e-role-management-token", token_type: "Bearer", principal: { principal_id: 131, principal_type: "staff", status: "active", school_id: 10 } } }); return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: { principal_id: 131, principal_type: "staff", school_id: 10, username: "e2e-role-management", permissions: ["roles.read","roles.create_custom","roles.update_custom","permissions.read","permissions.assign"] } }); return;
    }
    if (path === "/school/roles" && method === "GET") { await route.fulfill({ json: { roles } }); return; }
    if (path === "/school/roles" && method === "POST") {
      createCalls += 1;
      const body = request.postDataJSON() as Record<string, unknown>;
      if (failCreate) { failCreate = false; await route.fulfill({ status: 409, json: { status: "error", message: "A role with this name already exists.", field_errors: { name: "Choose a different role name." } } }); return; }
      const created = { id: 10, role_key: "assessment_coordinator", name: String(body.name), is_system: false, created_at: "2026-11-21T10:00:00Z" };
      roles = [...roles, created];
      await route.fulfill({ status: 201, json: { role: created } }); return;
    }
    if (path === "/school/permissions") {
      await route.fulfill({ json: { permissions: [
        { permission_key: "students.read_basic", description: "View basic student information.", is_delegable: true },
        { permission_key: "students.read_sensitive", description: "View sensitive student information.", is_delegable: true },
      ] } }); return;
    }
    if (path === "/school/permission-delegations") {
      await route.fulfill({ json: { delegations: [
        { id: 17, school_id: 10, permission_key: "students.read_basic", can_assign_to_roles: true, can_override_users: false, granted_by: 1, created_at: "2026-01-01T00:00:00Z" },
        { id: 18, school_id: 10, permission_key: "students.read_sensitive", can_assign_to_roles: false, can_override_users: false, granted_by: 1, created_at: "2026-01-01T00:00:00Z" },
      ] } }); return;
    }
    if (path === "/school/roles/10/permissions" && method === "GET") { await route.fulfill({ json: { permissions: grants } }); return; }
    if (path === "/school/roles/10/permissions" && method === "POST") {
      grantCalls += 1;
      const body = request.postDataJSON() as Record<string, unknown>;
      if (failGrant) { failGrant = false; await route.fulfill({ status: 409, json: { status: "error", message: "The delegated permission is no longer available.", field_errors: { permission_key: "Select an active permission delegated to this school." } } }); return; }
      grants = [{ permission_key: String(body.permission_key), description: "View basic student information.", delegation_id: Number(body.delegation_id), delegation_active: true, permission_active: true, created_at: "2026-11-21T10:00:00Z" }];
      await route.fulfill({ status: 201, json: { message: "Permission granted." } }); return;
    }
    if (path === "/school/roles/10/permissions/students.read_basic" && method === "DELETE") {
      revokeCalls += 1;
      if (failRevoke) { failRevoke = false; await route.fulfill({ status: 500, json: { status: "error", message: "Temporary permission removal failure." } }); return; }
      grants = [];
      await route.fulfill({ json: { message: "Permission grant removed." } }); return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-role-management");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Roles & permissions" }).click();

  await page.getByRole("button", { name: "Create role" }).click();
  const createDialog = page.getByRole("dialog", { name: "Create a custom role" });
  await createDialog.getByRole("button", { name: "Create role" }).click();
  await expect(createDialog.getByText("Enter a role name with at least 2 characters.")).toBeVisible();
  expect(createCalls).toBe(0);
  await createDialog.getByLabel("Role name").fill("Assessment Coordinator");
  await createDialog.getByRole("button", { name: "Create role" }).click();
  await expect(createDialog.getByText("Choose a different role name.")).toBeVisible();
  await expect(createDialog.getByLabel("Role name")).toHaveValue("Assessment Coordinator");
  expect(createCalls).toBe(1);

  await createDialog.getByRole("button", { name: "Create role" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Custom role created." })).toBeVisible();
  expect(createCalls).toBe(2);
  const permissionDialog = page.getByRole("dialog", { name: "Manage permissions · Assessment Coordinator" });
  await expect(permissionDialog.getByText("No permissions have been granted to this role yet.")).toBeVisible();

  await permissionDialog.getByRole("button", { name: "Add delegated permission" }).click();
  const grantDialog = page.getByRole("dialog", { name: "Grant a delegated permission" });
  await grantDialog.getByLabel("Permission to grant").click();
  await grantDialog.getByRole("option", { name: /students.read_basic/ }).click();
  await grantDialog.getByRole("button", { name: "Grant permission" }).click();
  await expect(grantDialog.getByText("Select an active permission delegated to this school.")).toBeVisible();
  expect(grantCalls).toBe(1);

  await grantDialog.getByLabel("Permission to grant").click();
  await grantDialog.getByRole("option", { name: /students.read_basic/ }).click();
  await grantDialog.getByRole("button", { name: "Grant permission" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Permission granted." })).toBeVisible();
  expect(grantCalls).toBe(2);
  await expect(permissionDialog.getByText("students.read_basic")).toBeVisible();
  await expect(permissionDialog.getByText("Active grant")).toBeVisible();

  await permissionDialog.getByRole("button", { name: "Remove students.read_basic" }).click();
  const revokeDialog = page.getByRole("dialog", { name: "Remove permission grant?" });
  await revokeDialog.getByRole("button", { name: "Cancel" }).click();
  await expect(revokeDialog).not.toBeVisible();
  expect(revokeCalls).toBe(0);

  await permissionDialog.getByRole("button", { name: "Remove students.read_basic" }).click();
  const retryDialog = page.getByRole("dialog", { name: "Remove permission grant?" });
  await retryDialog.getByRole("button", { name: "Remove permission" }).click();
  await expect(retryDialog.getByText("Temporary permission removal failure.")).toBeVisible();
  expect(revokeCalls).toBe(1);
  await retryDialog.getByRole("button", { name: "Remove permission" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Permission grant removed." })).toBeVisible();
  await expect(permissionDialog.getByText("No permissions have been granted to this role yet.")).toBeVisible();
  expect(revokeCalls).toBe(2);
});

test("permission delegation load errors are visible and retryable", async ({ page }) => {
  let failDelegations = true;
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/login" && method === "POST") { await route.fulfill({ json: { access_token: "e2e-role-retry-token", token_type: "Bearer", principal: { principal_id: 132, principal_type: "staff", status: "active", school_id: 10 } } }); return; }
    if (path === "/auth/me") { await route.fulfill({ json: { principal_id: 132, principal_type: "staff", school_id: 10, username: "e2e-role-retry", permissions: ["roles.read","roles.update_custom","permissions.read","permissions.assign"] } }); return; }
    if (path === "/school/roles") { await route.fulfill({ json: { roles: [{ id: 9, role_key: "library_coordinator", name: "Library Coordinator", is_system: false, created_at: "2026-01-01T00:00:00Z" }] } }); return; }
    if (path === "/school/permissions") { await route.fulfill({ json: { permissions: [{ permission_key: "students.read_basic", description: "View basic student information.", is_delegable: true }] } }); return; }
    if (path === "/school/permission-delegations") {
      if (failDelegations) { await route.fulfill({ status: 503, json: { message: "Delegation service temporarily unavailable." } }); return; }
      await route.fulfill({ json: { delegations: [{ id: 17, school_id: 10, permission_key: "students.read_basic", can_assign_to_roles: true, can_override_users: false, granted_by: 1, created_at: "2026-01-01T00:00:00Z" }] } }); return;
    }
    if (path === "/school/roles/9/permissions") { await route.fulfill({ json: { permissions: [] } }); return; }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-role-retry");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Roles & permissions" }).click();
  await page.getByRole("button", { name: "Manage permissions" }).click();
  const permissionDialog = page.getByRole("dialog", { name: "Manage permissions · Library Coordinator" });
  await expect(permissionDialog.getByRole("alert").filter({ hasText: "Delegation service temporarily unavailable." })).toBeVisible();
  await expect(permissionDialog.getByRole("button", { name: "Add delegated permission" })).toBeDisabled();
  failDelegations = false;
  await permissionDialog.getByRole("button", { name: "Try again" }).click();
  await expect(permissionDialog.getByRole("button", { name: "Add delegated permission" })).toBeEnabled();
});
