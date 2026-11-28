      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("staff-e2e-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.locator("button.nav-item").filter({ hasText: /^Staff$/ }).click();
}

test("staff workspace opens on a real-data overview and hides permission-gated or later-phase tabs", async ({ page }) => {