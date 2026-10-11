import { test, expect } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
const bindings = fs.readFileSync(
  path.resolve("bindings/github.com/SUDOKU-ASCII/sudoku-desktop/app.ts"),
  "utf8",
);
const ids = Object.fromEntries(
  [...bindings.matchAll(/export function (\w+)\([^]*?\$Call.ByID\((\d+)/g)].map(
    (m) => [Number(m[2]), m[1]],
  ),
);

test("import, connection state, settings drafts, routing and themes", async ({
  page,
}) => {
  let cfg: any = {
    version: 7,
    activeNodeId: "",
    nodes: [],
    routing: {
      proxyMode: "global",
      ruleUrls: [],
      customRulesEnabled: false,
      customRules: "",
    },
    tun: { enabled: true },
    core: { autoStart: false },
    ui: { theme: "auto", language: "ru", launchAtLogin: false },
    reverseClient: { routes: [] },
    reverseForward: {},
    portForwards: [],
  };
  let state: any = {
    running: false,
    coreRunning: false,
    tunRunning: false,
    startedAtUnix: 0,
    lastError: "previous shutdown notification",
    traffic: { recentBandwidth: [] },
    kernel: {},
    latencies: [],
    connections: [],
    recentLogs: [],
  };
  const errors: string[] = [];
  const calls: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/wails/runtime", async (route) => {
    const req = route.request().postDataJSON();
    let result: any = null;
    if (req.object === 0) {
      const method = ids[req.args.methodID],
        args = req.args.args || [];
      calls.push(method);
      if (method === "GetConfig") result = cfg;
      else if (method === "GetState") result = state;
      else if (method === "SaveConfig") cfg = args[0];
      else if (method === "ImportShortLink") {
        const n = {
          id: "node1",
          name: args[0] || "Мой сервер",
          serverAddress: "vpn.example.com:443",
          key: "test",
          sourceUrl: "https://example.com/sub/test.yaml",
          httpMask: {},
          customTables: [],
        };
        cfg.nodes = [n];
        cfg.activeNodeId = n.id;
        result = n;
      } else if (method === "StartProxy")
        state = {
          ...state,
          running: true,
          coreRunning: true,
          tunRunning: args[0].withTun,
          startedAtUnix: Math.floor(Date.now() / 1000),
        };
      else if (method === "StopProxy")
        state = {
          ...state,
          running: false,
          coreRunning: false,
          tunRunning: false,
        };
      else if (method === "DeleteNode") {
        cfg.nodes = [];
        cfg.activeNodeId = "";
      }
    }
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(result),
    });
  });
  await page.route("**/wails/custom.js", (r) =>
    r.fulfill({ status: 404, body: "" }),
  );
  await page.goto("/");
  await page.evaluate(() => document.fonts.ready);
  await assertFits(page);
  await expect(page.getByText("Добавьте сервер, чтобы начать")).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  state.lastError = "";
  await expect(
    page.getByRole("button", { name: "Подключить VPN", exact: true }),
  ).toBeDisabled();
  await page.getByRole("button", { name: /Добавить соединение/ }).click();
  await page
    .getByLabel("Ссылка соединения")
    .fill("https://example.com/sub/test.yaml");
  await page.getByLabel("Название").fill("Нидерланды");
  await page
    .getByRole("button", { name: "Добавить соединение", exact: true })
    .click();
  await expect(page.getByText("Нидерланды", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Подключить VPN", exact: true }).click();
  await expect(page.getByText("Соединение активно")).toBeVisible();
  await expect(page.locator(".power-button")).toHaveCount(1);
  await expect(page.locator(".page-home .primary")).toHaveCount(0);
  expect(state.tunRunning).toBe(true);
  await expect(page.locator("main")).not.toContainText(/\b\d{2}:\d{2}:\d{2}\b/);
  await page.getByRole("button", { name: "Настройки", exact: true }).click();
  await expect(
    page.getByRole("checkbox", { name: "VPN для всего устройства" }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("switch", { name: "VPN для всего устройства" }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Сохранить настройки" }),
  ).toBeEnabled();
  await page.getByRole("combobox", { name: "Режим" }).click();
  await page.getByRole("option", { name: "Все, кроме списка" }).click();
  calls.length = 0;
  await page.getByRole("button", { name: "Сохранить настройки" }).click();
  await expect(page.getByText("Соединение активно")).toBeVisible();
  expect(
    calls.filter((m) => ["StopProxy", "SaveConfig", "StartProxy"].includes(m)),
  ).toEqual(["StopProxy", "SaveConfig", "StartProxy"]);
  expect(state.tunRunning).toBe(true);
  expect(cfg.routing.proxyMode).toBe("pac");
  await page.getByRole("button", { name: "Настройки", exact: true }).click();
  await expect(page.getByLabel("Домены исключений")).toBeVisible();
  await page.setViewportSize({width: 328, height: 528});
  await assertFits(page);
  const hint = await page.locator('.reconnect-hint').boundingBox();
  const save = await page.getByRole('button', {name: 'Сохранить настройки'}).boundingBox();
  expect(save!.y - (hint!.y + hint!.height)).toBeGreaterThanOrEqual(14);
  await page.setViewportSize({width: 410, height: 660});
  await page.getByRole("button", { name: "Назад", exact: true }).click();
  await page.getByRole("button", { name: "Отключить VPN", exact: true }).click();
  await page.getByRole("button", { name: "Настройки", exact: true }).click();
  await assertFits(page);
  await page.getByRole("switch", { name: "Подключаться при запуске" }).check();
  await page.getByRole("switch", { name: "Запускать вместе с Windows" }).check();
  await page.getByRole("button", { name: "Тёмная", exact: true }).click();
  expect(cfg.core.autoStart).toBe(false);
  await page.getByRole("combobox", { name: "Режим" }).click();
  await page.getByRole("option", { name: "Все, кроме списка" }).click();
  for (const size of [
    { width: 410, height: 660 },
    { width: 360, height: 620 },
    { width: 328, height: 528 },
  ]) {
    await page.setViewportSize(size);
    await assertFits(page);
  }
  await page.setViewportSize({ width: 410, height: 660 });
  await page.getByLabel("Домены исключений").fill("https://bad.example");
  await page.getByRole("button", { name: "Сохранить настройки" }).click();
  await expect(page.getByRole("alert")).toContainText("Укажите домены");
  await page.getByLabel("Домены исключений").fill("hh.ru\navito.ru\nпример.рф");
  await page.getByRole("button", { name: "Сохранить настройки" }).click();
  await expect(page.getByRole("status")).toHaveText("Настройки сохранены");
  expect(cfg.core.autoStart).toBe(true);
  expect(cfg.ui.launchAtLogin).toBe(true);
  expect(cfg.routing.customRules).toContain("DOMAIN-SUFFIX,hh.ru,DIRECT");
  expect(cfg.routing.ruleUrls).toEqual([]);
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.evaluate(() => document.fonts.ready);
  expect(
    await page.locator("body").evaluate((e) => getComputedStyle(e).fontFamily),
  ).toContain("Montserrat");
  if (process.env.SUDOKU_PREVIEW_DIR) {
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      fullPage: true,
      path: path.join(process.env.SUDOKU_PREVIEW_DIR, "sudoku-dark.png"),
    });
    await page.getByRole("button", { name: "Настройки", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Сохранить настройки" }),
    ).toBeEnabled();
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      fullPage: true,
      path: path.join(process.env.SUDOKU_PREVIEW_DIR, "settings-dark.png"),
    });
    await page.getByRole("button", { name: "Светлая", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Сохранить настройки" }),
    ).toBeEnabled();
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      fullPage: true,
      path: path.join(process.env.SUDOKU_PREVIEW_DIR, "settings-light.png"),
    });
  }
  if ((await page.getByRole("combobox", { name: "Режим" }).count()) === 0) {
    await page.getByRole("button", { name: "Настройки", exact: true }).click();
  }
  for (const theme of ["Тёмная", "Светлая"]) {
    await page.getByRole("button", { name: theme, exact: true }).click();
    await page.getByRole("combobox", { name: "Режим" }).click();
    await expect(page.getByRole("listbox")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Сохранить настройки" }),
    ).toHaveCSS("opacity", "1");
    await expect
      .poll(() =>
        page
          .getByRole("combobox", { name: "Режим" })
          .evaluate(
            (e) =>
              getComputedStyle(e).backgroundColor ===
              getComputedStyle(document.getElementById("routing-options")!)
                .backgroundColor,
          ),
      )
      .toBe(true);
    const colours = await page
      .getByRole("option")
      .first()
      .evaluate((e) => ({
        text: getComputedStyle(e).color,
        background: getComputedStyle(e.parentElement!).backgroundColor,
      }));
    expect(colours.text).not.toBe(colours.background);
    if (process.env.SUDOKU_PREVIEW_DIR)
      await page.screenshot({
        path: path.join(
          process.env.SUDOKU_PREVIEW_DIR,
          `settings-menu-${theme === "Тёмная" ? "dark" : "light"}.png`,
        ),
      });
    await page.getByRole("combobox", { name: "Режим" }).press("Escape");
    await expect(page.getByRole("listbox")).toHaveCount(0);
    await expect(page.locator(".toggle-row input").first()).toHaveCSS(
      "opacity",
      "0",
    );
    await page.evaluate(() => {
      document
        .querySelectorAll(".toggle-row input")
        .forEach((e) => ((e as HTMLInputElement).disabled = true));
    });
    await expect(page.locator(".toggle-row input").first()).toHaveCSS(
      "opacity",
      "0",
    );
    await page.evaluate(() => {
      document
        .querySelectorAll(".toggle-row input")
        .forEach((e) => ((e as HTMLInputElement).disabled = false));
    });
  }
  expect(errors).toEqual([]);
});

async function assertFits(page: import("@playwright/test").Page) {
  await expect(page.locator("main")).toBeVisible();
  await page.evaluate(() => document.fonts.ready);
  const metrics = await page.evaluate(() => {
    const main = document.querySelector("main")!;
    const footer = document.querySelector("footer")!.getBoundingClientRect();
    const controls = [
      ...main.querySelectorAll("button,select,textarea,input"),
    ].filter((e) => e.getBoundingClientRect().height);
    return {
      viewport: innerHeight,
      root: document.documentElement.scrollHeight,
      width: document.documentElement.scrollWidth,
      viewportWidth: innerWidth,
      footerBottom: footer.bottom,
      lastControl: Math.max(
        ...controls.map((e) => e.getBoundingClientRect().bottom),
      ),
      footerTop: footer.top,
    };
  });
  expect(metrics.root).toBeLessThanOrEqual(metrics.viewport);
  expect(metrics.width).toBeLessThanOrEqual(metrics.viewportWidth);
  expect(metrics.footerBottom).toBeLessThanOrEqual(metrics.viewport + 1);
  expect(metrics.lastControl).toBeLessThanOrEqual(metrics.footerTop);
  await expect(page.locator("main")).not.toContainText(
    /ЛИЧНОЕ СОЕДИНЕНИЕ|Свобода быть онлайн|Ваше спокойное пространство|Для VPN запустите приложение/,
  );
}

test('protocol buttons switch saved profiles during a connection',async({page})=>{
 let cfg:any={version:7,activeNodeId:'s',nodes:[{id:'s',name:'Sudoku server',protocol:'sudoku',serverAddress:'s.example:443'},{id:'x',name:'Xray server',protocol:'xray',serverAddress:'x.example:443'},{id:'m',name:'Mist server',protocol:'mist',serverAddress:'m.example:443'}],routing:{proxyMode:'global'},tun:{enabled:true},core:{autoStart:false},ui:{theme:'light'}};
 let state:any={running:true,coreRunning:true,tunRunning:true,traffic:{recentBandwidth:[]},kernel:{},recentLogs:[]};const switched:string[]=[];
 await page.route('**/wails/runtime',async route=>{const req=route.request().postDataJSON();let result:any=null;if(req.object===0){const method=ids[req.args.methodID];if(method==='GetConfig')result=cfg;else if(method==='GetState')result=state;else if(method==='SwitchNode'){cfg.activeNodeId=req.args.args[0];switched.push(cfg.activeNodeId)}}await route.fulfill({contentType:'application/json',body:JSON.stringify(result)})});
 await page.goto('/');await expect(page.getByText('Sudoku server',{exact:true})).toBeVisible();for(const [protocol,name]of [['Xray','Xray server'],['Mist','Mist server'],['Sudoku','Sudoku server']]){await page.getByRole('navigation',{name:'Протокол'}).getByRole('button',{name:protocol,exact:true}).click();await expect(page.getByText(name,{exact:true})).toBeVisible();await expect(page.getByText('Соединение активно')).toBeVisible();await assertFits(page)}await page.getByRole('navigation',{name:'Протокол'}).getByRole('button',{name:'Sudoku',exact:true}).click();expect(switched).toEqual(['x','m','s']);
});
