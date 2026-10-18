"use strict";

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const origDefineProperty = Object.defineProperty;
const registry = new Map();

function writeStdoutSync(str) {
  const buf = Buffer.from(str, "utf8");
  let off = 0;
  while (off < buf.length) {
    try {
      const n = fs.writeSync(1, buf, off, buf.length - off);
      if (n > 0) {
        off += n;
      }
    } catch (err) {
      if (err && (err.code === "EAGAIN" || err.code === "EWOULDBLOCK")) {
        continue;
      }
      throw err;
    }
  }
}

function toMsString(val) {
  if (val === null || val === undefined) {
    return null;
  }
  if (typeof val === "number" && Number.isFinite(val)) {
    return String(Math.trunc(val));
  }
  if (typeof val === "string" && val.trim() !== "") {
    return val.trim();
  }
  return null;
}

function classifyErrorCode(err) {
  if (!err) {
    return "unknown";
  }
  if (typeof err.httpStatus === "number") {
    return "http_" + String(err.httpStatus);
  }
  if (typeof err.status === "number") {
    return "http_" + String(err.status);
  }
  const code = typeof err.code === "string" ? err.code : "";
  const msg = typeof err.message === "string" ? err.message : "";
  const combined = (code + " " + msg).toLowerCase();
  if (
    combined.includes("project_token_login_required") ||
    combined.includes("project_token_account_required") ||
    combined.includes("project_token_scope_invalidated") ||
    combined.includes("unauthenticated") ||
    combined.includes("unauthorized") ||
    combined.includes("401") ||
    combined.includes("403")
  ) {
    return "auth_required";
  }
  if (combined.includes("429") || combined.includes("rate_limit")) {
    return "rate_limited";
  }
  if (combined.includes("timed out") || combined.includes("timeout")) {
    return "timeout";
  }
  if (code) {
    return code;
  }
  return "unknown";
}

function serializeError(err) {
  return {
    name: err && typeof err.name === "string" ? err.name : "Error",
    code: classifyErrorCode(err),
  };
}

function isWriteOpenFlags(flags) {
  if (typeof flags === "string") {
    return (
      flags.includes("w") ||
      flags.includes("a") ||
      flags.includes("+") ||
      flags.includes("x")
    );
  }
  if (typeof flags === "number") {
    const c = fs.constants || {};
    const mask =
      (c.O_WRONLY || 1) |
      (c.O_RDWR || 2) |
      (c.O_CREAT || 64) |
      (c.O_TRUNC || 512) |
      (c.O_APPEND || 1024);
    return (flags & mask) !== 0;
  }
  return false;
}

function enableReadOnlyFsGuard(allowedDir) {
  const allowedRoots = [path.resolve(allowedDir)];
  try {
    allowedRoots.push(fs.realpathSync(allowedDir));
  } catch {
    // Ignore if realpathSync fails.
  }

  function isPathAllowed(target) {
    if (typeof target === "number") {
      return false;
    }
    let str = "";
    if (typeof target === "string") {
      str = target;
    } else if (Buffer.isBuffer(target)) {
      str = target.toString("utf8");
    } else if (target && typeof target.pathname === "string") {
      str = target.pathname;
    } else {
      return false;
    }
    if (!str) {
      return false;
    }
    const resolved = path.resolve(str);
    return allowedRoots.some(
      (root) => resolved === root || resolved.startsWith(root + path.sep)
    );
  }

  function makeRoError(op) {
    const err = new Error("EROFS: read-only file system, " + op);
    err.code = "EROFS";
    return err;
  }

  const singlePathSyncOps = [
    "writeFileSync",
    "appendFileSync",
    "mkdirSync",
    "chmodSync",
    "unlinkSync",
    "rmSync",
    "rmdirSync",
    "truncateSync",
  ];
  for (const op of singlePathSyncOps) {
    if (typeof fs[op] === "function") {
      const orig = fs[op];
      fs[op] = function (target) {
        if (!isPathAllowed(target)) {
          throw makeRoError(op);
        }
        return orig.apply(this, arguments);
      };
    }
  }

  const dualPathSyncOps = ["renameSync", "copyFileSync"];
  for (const op of dualPathSyncOps) {
    if (typeof fs[op] === "function") {
      const orig = fs[op];
      fs[op] = function (src, dest) {
        if (!isPathAllowed(src) || !isPathAllowed(dest)) {
          throw makeRoError(op);
        }
        return orig.apply(this, arguments);
      };
    }
  }

  if (typeof fs.openSync === "function") {
    const origOpenSync = fs.openSync;
    fs.openSync = function (target, flags) {
      if (isWriteOpenFlags(flags) && !isPathAllowed(target)) {
        throw makeRoError("openSync");
      }
      return origOpenSync.apply(this, arguments);
    };
  }

  if (typeof fs.createWriteStream === "function") {
    const origCreateWriteStream = fs.createWriteStream;
    fs.createWriteStream = function (target) {
      if (!isPathAllowed(target)) {
        throw makeRoError("createWriteStream");
      }
      return origCreateWriteStream.apply(this, arguments);
    };
  }

  const singlePathAsyncOps = [
    "writeFile",
    "appendFile",
    "mkdir",
    "chmod",
    "unlink",
    "rm",
    "rmdir",
    "truncate",
  ];
  for (const op of singlePathAsyncOps) {
    if (typeof fs[op] === "function") {
      const orig = fs[op];
      fs[op] = function (target) {
        if (!isPathAllowed(target)) {
          const err = makeRoError(op);
          const cb =
            arguments.length > 0 &&
            typeof arguments[arguments.length - 1] === "function"
              ? arguments[arguments.length - 1]
              : null;
          if (cb) {
            process.nextTick(cb, err);
            return;
          }
          throw err;
        }
        return orig.apply(this, arguments);
      };
    }
  }

  const dualPathAsyncOps = ["rename", "copyFile"];
  for (const op of dualPathAsyncOps) {
    if (typeof fs[op] === "function") {
      const orig = fs[op];
      fs[op] = function (src, dest) {
        if (!isPathAllowed(src) || !isPathAllowed(dest)) {
          const err = makeRoError(op);
          const cb =
            arguments.length > 0 &&
            typeof arguments[arguments.length - 1] === "function"
              ? arguments[arguments.length - 1]
              : null;
          if (cb) {
            process.nextTick(cb, err);
            return;
          }
          throw err;
        }
        return orig.apply(this, arguments);
      };
    }
  }

  if (typeof fs.open === "function") {
    const origOpenCb = fs.open;
    fs.open = function (target, flags) {
      const effectiveFlags = typeof flags === "function" ? "r" : flags;
      if (isWriteOpenFlags(effectiveFlags) && !isPathAllowed(target)) {
        const err = makeRoError("open");
        const cb =
          arguments.length > 0 &&
          typeof arguments[arguments.length - 1] === "function"
            ? arguments[arguments.length - 1]
            : null;
        if (cb) {
          process.nextTick(cb, err);
          return;
        }
        throw err;
      }
      return origOpenCb.apply(this, arguments);
    };
  }

  let fsPromises = null;
  try {
    fsPromises = require("node:fs/promises");
  } catch {
    fsPromises = fs.promises || null;
  }
  if (fsPromises) {
    for (const op of singlePathAsyncOps) {
      if (typeof fsPromises[op] === "function") {
        const orig = fsPromises[op];
        fsPromises[op] = async function (target) {
          if (!isPathAllowed(target)) {
            throw makeRoError(op);
          }
          return orig.apply(this, arguments);
        };
      }
    }
    for (const op of dualPathAsyncOps) {
      if (typeof fsPromises[op] === "function") {
        const orig = fsPromises[op];
        fsPromises[op] = async function (src, dest) {
          if (!isPathAllowed(src) || !isPathAllowed(dest)) {
            throw makeRoError(op);
          }
          return orig.apply(this, arguments);
        };
      }
    }
    if (typeof fsPromises.open === "function") {
      const origOpen = fsPromises.open;
      fsPromises.open = async function (target, flags) {
        if (isWriteOpenFlags(flags) && !isPathAllowed(target)) {
          throw makeRoError("open");
        }
        return origOpen.apply(this, arguments);
      };
    }
  }
}

function validateCredentialFileBeforeRead(filePath) {
  if (typeof filePath !== "string" || !filePath.trim()) {
    return;
  }
  let raw;
  try {
    raw = fs.readFileSync(filePath, "utf8");
  } catch (err) {
    if (err && err.code === "ENOENT") {
      return;
    }
    throw err;
  }
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    const err = new Error("Shared ZCode credentials are corrupt");
    err.code = "corrupt_credentials";
    throw err;
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    const err = new Error("Credential record must be an object");
    err.code = "corrupt_credentials";
    throw err;
  }
  for (const [k, v] of Object.entries(parsed)) {
    if (typeof v !== "string") {
      const err = new Error("Credential record value must be a string: " + k);
      err.code = "corrupt_credentials";
      throw err;
    }
  }
}

function createReadOnlyCredentialStore(rawStore) {
  const memory = new Map();
  const deleted = new Set();
  let validated = false;
  function ensureValidOnDisk() {
    if (!validated) {
      validateCredentialFileBeforeRead(rawStore && rawStore.filePath);
      validated = true;
    }
  }
  return {
    filePath: rawStore.filePath,
    async load(key) {
      if (deleted.has(key)) {
        return null;
      }
      if (memory.has(key)) {
        return memory.get(key);
      }
      ensureValidOnDisk();
      return rawStore.load(key);
    },
    async loadMany(keys) {
      const missing = [];
      const out = {};
      for (const k of keys || []) {
        if (deleted.has(k)) {
          out[k] = null;
        } else if (memory.has(k)) {
          out[k] = memory.get(k);
        } else {
          missing.push(k);
        }
      }
      if (missing.length > 0) {
        ensureValidOnDisk();
        const loaded = await rawStore.loadMany(missing);
        for (const k of missing) {
          out[k] = loaded && loaded[k] !== undefined ? loaded[k] : null;
        }
      }
      return out;
    },
    async save(key, value) {
      deleted.delete(key);
      memory.set(key, value);
    },
    async saveMany(entries) {
      if (!entries || typeof entries !== "object") {
        return;
      }
      for (const [k, v] of Object.entries(entries)) {
        deleted.delete(k);
        memory.set(k, v);
      }
    },
    async delete(key) {
      memory.delete(key);
      deleted.add(key);
    },
    async deleteMany(keys) {
      for (const k of keys || []) {
        memory.delete(k);
        deleted.add(k);
      }
    },
    async deleteIfValues(entries) {
      if (!entries || typeof entries !== "object") {
        return;
      }
      for (const k of Object.keys(entries)) {
        memory.delete(k);
        deleted.add(k);
      }
    },
  };
}

function createReadOnlyQuotaHttpClient(rawHttpClient) {
  return {
    async request(req) {
      const method =
        req && typeof req.method === "string"
          ? req.method.trim().toUpperCase()
          : "GET";
      if (method !== "GET") {
        const err = new Error("non-GET request blocked in read-only mode");
        err.code = "read_only_violation";
        throw err;
      }
      let parsedUrl;
      try {
        parsedUrl = new URL(String((req && req.url) || ""));
      } catch {
        const err = new Error("invalid quota request URL");
        err.code = "read_only_violation";
        throw err;
      }
      if (parsedUrl.pathname !== "/api/monitor/usage/quota/limit") {
        const err = new Error(
          "unexpected request path blocked in read-only mode"
        );
        err.code = "read_only_violation";
        throw err;
      }
      return rawHttpClient.request({
        method: "GET",
        url: req.url,
        headers: req.headers,
        timeoutMs: req.timeoutMs,
        signal: req.signal,
      });
    },
  };
}

function readPreferredProviderId(personalPath) {
  try {
    const raw = fs.readFileSync(personalPath, "utf8");
    const parsed = JSON.parse(raw);
    const sel =
      (parsed && parsed.config && parsed.config.defaultModelSelection) ||
      (parsed && parsed.defaultModelSelection) ||
      null;
    if (sel && typeof sel.providerId === "string" && sel.providerId.trim()) {
      return sel.providerId.trim();
    }
  } catch {
    // Missing or unreadable personal provider config falls back to store state.
  }
  return null;
}

function resolveBuiltinProviderConfigPath(dataBaseDir, getSeaAssets) {
  if (process.argv[1]) {
    try {
      const entryDir = path.dirname(
        fs.realpathSync(path.resolve(process.argv[1]))
      );
      const candidates = [
        path.join(entryDir, "provider", "zcode-builtin.json"),
        path.resolve(entryDir, "../config/provider/zcode-builtin.json"),
        path.resolve(entryDir, "../../config/provider/zcode-builtin.json"),
        path.resolve(
          entryDir,
          "../../../../../config/provider/zcode-builtin.json"
        ),
      ];
      const found = candidates.find((p) => fs.existsSync(p));
      if (found) {
        return found;
      }
    } catch {
      // Ignore realpath resolution errors and check existing read-only cache or SEA assets.
    }
  }
  const existingCacheCandidates = [
    path.join(
      dataBaseDir,
      ".zcode",
      "v2",
      "runtime",
      "provider",
      "active",
      "zcode-builtin.json"
    ),
    path.join(
      dataBaseDir,
      ".zcode",
      "v2",
      "runtime",
      "provider",
      "bundled",
      "zcode-builtin.json"
    ),
  ];
  const existingCache = existingCacheCandidates.find((p) => fs.existsSync(p));
  if (existingCache) {
    return existingCache;
  }
  if (typeof getSeaAssets === "function") {
    try {
      const sea = getSeaAssets();
      if (
        sea &&
        typeof sea.isSea === "function" &&
        sea.isSea() &&
        typeof sea.getAsset === "function"
      ) {
        const content = sea.getAsset(
          "zcode-provider/zcode-builtin.json",
          "utf8"
        );
        if (typeof content === "string" && content.trim()) {
          const tempBuiltinPath = path.join(__dirname, "zcode-builtin.json");
          fs.writeFileSync(tempBuiltinPath, content, {
            encoding: "utf8",
            mode: 0o600,
          });
          return tempBuiltinPath;
        }
      }
    } catch {
      // Fall back to empty path if SEA asset extraction is unavailable.
    }
  }
  return "";
}

function sanitizeLimits(rawLimits) {
  if (!Array.isArray(rawLimits)) {
    return null;
  }
  return rawLimits
    .filter((item) => item && typeof item === "object")
    .map((item) => ({
      type: typeof item.type === "string" ? item.type : null,
      unit:
        typeof item.unit === "number" && Number.isFinite(item.unit)
          ? item.unit
          : null,
      number:
        typeof item.number === "number" && Number.isFinite(item.number)
          ? item.number
          : null,
      percentage:
        typeof item.percentage === "number" && Number.isFinite(item.percentage)
          ? item.percentage
          : null,
      usage:
        typeof item.usage === "number" && Number.isFinite(item.usage)
          ? item.usage
          : null,
      currentValue:
        typeof item.currentValue === "number" &&
        Number.isFinite(item.currentValue)
          ? item.currentValue
          : null,
      remaining:
        typeof item.remaining === "number" && Number.isFinite(item.remaining)
          ? item.remaining
          : null,
      nextResetTime: toMsString(item.nextResetTime),
    }));
}

function wrapRunDoctor(origFn) {
  const wrapped = async function (io, options) {
    if (!options || !options.json) {
      return origFn.apply(this, arguments);
    }
    const loadBootstrap = registry.get("loadBootstrapModule");
    if (typeof loadBootstrap === "function") {
      try {
        await loadBootstrap();
      } catch {
        // Proceed to function availability check below.
      }
    }

    enableReadOnlyFsGuard(__dirname);

    const createCredentialStore = registry.get(
      "createSharedZCodeCredentialStore"
    );
    const readProviders = registry.get("readStandaloneCodingPlanProviders");
    const readSnapshot = registry.get(
      "readStandaloneAccountProviderConfigSnapshot"
    );
    const createHttpClient = registry.get("createDefaultHttpClient");
    const resolveEndpoints = registry.get(
      "resolveRuntimeProductEndpointConfig"
    );
    const getSeaAssets = registry.get("getSeaProviderConfigAssets");

    if (
      typeof createCredentialStore !== "function" ||
      typeof readProviders !== "function" ||
      typeof readSnapshot !== "function" ||
      typeof createHttpClient !== "function" ||
      typeof resolveEndpoints !== "function"
    ) {
      writeStdoutSync(JSON.stringify({ status: "unsupported" }) + "\n");
      process.exit(0);
    }

    try {
      const env = Object.assign({}, process.env);
      const dataBaseDir =
        (typeof env.ZCODE_DATA_BASE_DIR === "string" &&
          env.ZCODE_DATA_BASE_DIR.trim()) ||
        os.homedir();
      const personalPath =
        (typeof env.ZCODE_PERSONAL_PROVIDER_CONFIG_FILE === "string" &&
          env.ZCODE_PERSONAL_PROVIDER_CONFIG_FILE.trim()) ||
        path.join(dataBaseDir, ".zcode", "v2", "provider_config.json");
      let builtinPath =
        typeof env.ZCODE_BUILTIN_PROVIDER_CONFIG_FILE === "string" &&
        env.ZCODE_BUILTIN_PROVIDER_CONFIG_FILE.trim()
          ? env.ZCODE_BUILTIN_PROVIDER_CONFIG_FILE.trim()
          : "";
      if (!builtinPath) {
        builtinPath = resolveBuiltinProviderConfigPath(
          dataBaseDir,
          getSeaAssets
        );
      }
      env.ZCODE_BUILTIN_PROVIDER_CONFIG_FILE = builtinPath;
      env.ZCODE_PERSONAL_PROVIDER_CONFIG_FILE = personalPath;

      const rawStore = createCredentialStore({ env });
      const store = createReadOnlyCredentialStore(rawStore);
      const catalog = await readProviders(env);
      const providerApiKeys = new Map();
      for (const item of Array.isArray(catalog) ? catalog : []) {
        if (
          !item ||
          typeof item.providerId !== "string" ||
          typeof item.family !== "string"
        ) {
          continue;
        }
        const providerId = item.providerId.trim();
        const idKey = "account-provider:" + providerId + ":identity";
        const sourceKey = "account-provider:" + providerId + ":auth-source";
        let accountIdentity = "";
        const existingId = await store.load(idKey);
        const existingSource = await store.load(sourceKey);
        const existingSourceTrimmed =
          typeof existingSource === "string" ? existingSource.trim() : "";
        if (existingId && String(existingId).trim()) {
          accountIdentity = String(existingId).trim();
        } else {
          try {
            const rawUser = await store.load(
              "oauth:" + item.family + ":user_info"
            );
            if (rawUser) {
              const parsedUser = JSON.parse(String(rawUser));
              const candidate =
                (parsedUser && (parsedUser.id || parsedUser.user_id)) || "";
              if (String(candidate).trim()) {
                accountIdentity = String(candidate).trim();
              }
            }
          } catch {
            // Fall back below if user_info is not valid JSON.
          }
          if (!accountIdentity) {
            const oauthToken = await store.load(
              "oauth:" + item.family + ":access_token"
            );
            if (oauthToken && String(oauthToken).trim()) {
              accountIdentity = item.family;
            }
          }
        }

        let cachedKey = "";
        if (accountIdentity) {
          const apiKeySlot =
            "account-provider:coding-plan:" +
            providerId +
            ":account:" +
            encodeURIComponent(accountIdentity) +
            ":api-key";
          const rawKey = await store.load(apiKeySlot);
          if (rawKey && String(rawKey).trim()) {
            cachedKey = String(rawKey).trim();
          }
        }

        if (cachedKey) {
          providerApiKeys.set(providerId, cachedKey);
        }

        if (accountIdentity) {
          await store.save(idKey, accountIdentity);
          await store.save(
            sourceKey,
            existingSourceTrimmed || (cachedKey ? "manual" : "oauth")
          );
        }
      }

      const snapshot = await readSnapshot(store, env);

      const catalogList = Array.isArray(catalog) ? catalog : [];
      const entitled = catalogList.filter((item) => {
        if (!item || typeof item.providerId !== "string") {
          return false;
        }
        const providerId = item.providerId.trim();
        const rule =
          snapshot &&
          snapshot.providers &&
          typeof snapshot.providers.get === "function"
            ? snapshot.providers.get(providerId)
            : null;
        return Boolean(rule && rule.access && rule.access.entitled === true);
      });
      const entitledIds = new Set(
        entitled.map((item) => item.providerId.trim())
      );

      const preferredProviderId = readPreferredProviderId(personalPath);
      const activeFamilyRaw = await Promise.resolve(
        store.load("oauth:active_provider")
      ).catch(() => null);
      const activeFamily =
        typeof activeFamilyRaw === "string" ? activeFamilyRaw.trim() : "";

      let selected = null;
      if (preferredProviderId) {
        selected =
          entitled.find((p) => p.providerId === preferredProviderId) ||
          catalogList.find((p) => p.providerId === preferredProviderId) ||
          null;
      }
      if (!selected && activeFamily) {
        selected =
          entitled.find((p) => p.family === activeFamily) ||
          catalogList.find((p) => p.family === activeFamily) ||
          null;
      }
      if (!selected) {
        selected = entitled[0] || null;
      }

      const selectedProviderId =
        selected && typeof selected.providerId === "string"
          ? selected.providerId.trim()
          : "";
      const apiKey = selectedProviderId
        ? providerApiKeys.get(selectedProviderId) || ""
        : "";
      if (!selectedProviderId || !entitledIds.has(selectedProviderId) || !apiKey) {
        writeStdoutSync(
          JSON.stringify({
            status: "unauthenticated",
            isAuthenticated: false,
          }) + "\n"
        );
        process.exit(0);
      }

      const endpoints = resolveEndpoints(env);
      const origin =
        selected.family === "bigmodel"
          ? endpoints.bigModelApiOrigin
          : endpoints.zaiBusinessBaseUrl;
      const url =
        String(origin).replace(/\/+$/, "") + "/api/monitor/usage/quota/limit";
      const httpClient = createReadOnlyQuotaHttpClient(createHttpClient(env));

      let response;
      try {
        response = await httpClient.request({
          method: "GET",
          url,
          headers: {
            Accept: "application/json",
            Authorization: apiKey,
          },
        });
      } catch (err) {
        writeStdoutSync(
          JSON.stringify({
            status: "authenticated",
            isAuthenticated: true,
            family: selected.family,
            quota: null,
            quotaError: serializeError(err),
          }) + "\n"
        );
        process.exit(0);
      }

      const httpStatus =
        response && typeof response.status === "number"
          ? response.status
          : null;
      let parsedBody = null;
      try {
        const bodyText =
          typeof response.body === "string"
            ? response.body
            : response &&
                (Buffer.isBuffer(response.body) ||
                  response.body instanceof Uint8Array)
              ? Buffer.from(response.body).toString("utf8")
              : String((response && response.body) || "");
        parsedBody = JSON.parse(bodyText);
      } catch {
        parsedBody = null;
      }

      const quota = parsedBody
        ? {
            httpStatus,
            code:
              typeof parsedBody.code === "number" ? parsedBody.code : null,
            success:
              typeof parsedBody.success === "boolean"
                ? parsedBody.success
                : null,
            limits:
              parsedBody.data && typeof parsedBody.data === "object"
                ? sanitizeLimits(parsedBody.data.limits)
                : null,
          }
        : {
            httpStatus,
            code: null,
            success: null,
            limits: null,
          };

      writeStdoutSync(
        JSON.stringify({
          status: "authenticated",
          isAuthenticated: true,
          family: selected.family,
          quota,
          quotaError: null,
        }) + "\n"
      );
      process.exit(0);
    } catch (err) {
      writeStdoutSync(
        JSON.stringify({
          status: "authenticated",
          isAuthenticated: true,
          family: null,
          quota: null,
          quotaError: serializeError(err),
        }) + "\n"
      );
      process.exit(0);
    }
  };
  wrapped.__gaebokchiWrapped = true;
  return wrapped;
}

Object.defineProperty = function (target, prop, descriptor) {
  if (
    typeof target === "function" &&
    prop === "name" &&
    descriptor &&
    typeof descriptor.value === "string"
  ) {
    const fnName = descriptor.value;
    if (fnName === "runDoctor" && !target.__gaebokchiWrapped) {
      const wrapped = wrapRunDoctor(target);
      origDefineProperty.call(Object, wrapped, prop, descriptor);
      registry.set(fnName, wrapped);
      return wrapped;
    }
    registry.set(fnName, target);
  }
  return origDefineProperty.apply(this, arguments);
};
