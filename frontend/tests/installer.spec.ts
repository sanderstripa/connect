import { test, expect } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
const root = path.resolve("../cmd/installer/web");

for (const uninstall of [false, true]) {
  test(`installer themes, languages and ${uninstall ? "removal" : "installation"}`, async ({
    page,
  }) => {
    const calls: string[] = [];
    let fail = true;
    await page.route("**/setup-test/**", async (route) => {
      let name =
        new URL(route.request().url()).pathname.replace("/setup-test/", "") ||
        "index.html";
      const file = path.resolve(root, name);
      if (!file.startsWith(root + path.sep)) throw Error("invalid asset");
      let data = fs.readFileSync(file);
      if (name === "index.html")
        data = Buffer.from(
          data
            .toString()
            .replace("__TOKEN__", "test-token")
            .replace("__MODE__", String(uninstall)),
        );
      const type = name.endsWith(".js")
        ? "application/javascript"
        : name.endsWith(".css")
          ? "text/css"
          : name.endsWith(".png")
            ? "image/png"
            : name.endsWith(".ttf")
              ? "font/ttf"
              : "text/html";
      await route.fulfill({ body: data, contentType: type });
    });
    await page.route("**/api/**", async (route) => {
      expect(route.request().headers()["x-setup-token"]).toBe("test-token");
      const method = new URL(route.request().url()).pathname;
      calls.push(method);
      if (method === "/api/install") {
        expect(route.request().postDataJSON()).toEqual({ desktop: true, chrome: true });
      }
      await route.fulfill({
        status: fail ? 500 : 200,
        contentType: "application/json",
        body: JSON.stringify(fail ? { error: "busy app" } : { ok: true }),
      });
    });
    await page.setViewportSize({ width: 440, height: 600 });
    await page.goto("/setup-test/");
    await page.evaluate(() => document.fonts.ready);
    if ((await page.locator("#language").innerText()) === "EN")
      await page.locator("#language").click();
    await expect(page.locator("#title")).toHaveText(
      uninstall ? "Удаление Connect" : "Установка Connect",
    );
    await page.locator("#language").click();
    await expect(page.locator("#language")).toHaveText("EN");
    await expect(page.locator("#primary")).toHaveText(
      uninstall ? "Uninstall" : "Install",
    );
    await page.locator("#language").click();
    await expect(page.locator("#language")).toHaveText("RU");
    for (const theme of ["light", "dark"]) {
      if ((await page.locator("html").getAttribute("data-theme")) !== theme)
        await page.locator("#theme").click();
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollHeight <= innerHeight,
        ),
      ).toBe(true);
      if (process.env.SUDOKU_PREVIEW_DIR)
        await page.screenshot({
          path: path.join(
            process.env.SUDOKU_PREVIEW_DIR,
            `setup-${theme}${uninstall ? "-uninstall" : ""}.png`,
          ),
        });
    }
    await page.locator("#primary").click();
    await expect(page.locator("#error")).toBeVisible();
    await expect(page.locator("#primary")).toBeEnabled();
    fail = false;
    await page.locator("#primary").click();
    await expect(page.locator("#title")).toHaveText(
      uninstall ? "Connect удалён" : "Connect установлен",
    );
    if (!uninstall) {
      await expect.poll(() => calls.filter(c => c === "/api/chrome").length).toBe(1);
      await expect(page.locator("#chromeNote")).toContainText("Ctrl+V");
      await page.locator("#chromeSetup").click();
      await expect.poll(() => calls.filter(c => c === "/api/chrome").length).toBe(2);
      expect(await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight)).toBe(true);
    }
    await page.locator("#primary").click();
    await expect.poll(() => calls).toContain("/api/close");
    if (!uninstall) expect(calls).toContain("/api/open");
  });
}
