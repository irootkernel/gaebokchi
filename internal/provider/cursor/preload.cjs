"use strict";

const Module = require("node:module");
const fs = require("node:fs");

const origLoad = Module._load;

function serializeError(err) {
  if (!err) {
    return { name: "Error", code: "unknown" };
  }
  return {
    name: typeof err.name === "string" ? err.name : "Error",
    code: err.code !== undefined ? String(err.code) : "unknown",
  };
}

function toMsString(val) {
  if (val === null || val === undefined) {
    return null;
  }
  return String(val);
}

function wrapHandleStatus(origFn) {
  const wrapped = async function (credentialManager, dashboardClient, options) {
    if (options && options.format && options.format !== "json") {
      return origFn.apply(this, arguments);
    }
    const accessToken = await Promise.resolve(
      credentialManager.getAccessToken()
    ).catch(() => null);
    const refreshToken = await Promise.resolve(
      credentialManager.getRefreshToken()
    ).catch(() => null);
    const apiKey = await Promise.resolve(
      credentialManager.getApiKey()
    ).catch(() => null);

    const hasAccessToken = Boolean(accessToken);
    const hasRefreshToken = Boolean(refreshToken);
    const hasApiKey = Boolean(apiKey);
    const isAuthenticated = Boolean(accessToken || apiKey);

    if (hasApiKey) {
      fs.writeSync(
        1,
        JSON.stringify({
          status: "api_key",
          isAuthenticated: true,
          hasAccessToken: hasAccessToken,
          hasRefreshToken: hasRefreshToken,
          hasApiKey: true,
        }) + "\n"
      );
      return 0;
    }

    if (!isAuthenticated || !hasAccessToken || !hasRefreshToken) {
      fs.writeSync(
        1,
        JSON.stringify({
          status: "unauthenticated",
          isAuthenticated: isAuthenticated,
          hasAccessToken: hasAccessToken,
          hasRefreshToken: hasRefreshToken,
          hasApiKey: false,
        }) + "\n"
      );
      return 0;
    }

    const [planRes, usageRes] = await Promise.allSettled([
      dashboardClient.getPlanInfo({}),
      dashboardClient.getCurrentPeriodUsage({}),
    ]);

    let planInfo = null;
    let planError = null;
    if (planRes.status === "fulfilled" && planRes.value && planRes.value.planInfo) {
      const p = planRes.value.planInfo;
      planInfo = {
        planName: typeof p.planName === "string" ? p.planName : null,
        includedUsagePeriod:
          typeof p.includedUsagePeriod === "number" ? p.includedUsagePeriod : null,
        includedUsageResetsAtMs: toMsString(p.includedUsageResetsAt),
        billingCycleEndMs: toMsString(p.billingCycleEnd),
      };
    } else if (planRes.status === "rejected") {
      planError = serializeError(planRes.reason);
    }

    let currentPeriodUsage = null;
    let usageError = null;
    if (usageRes.status === "fulfilled" && usageRes.value) {
      const u = usageRes.value;
      const pu = u.planUsage || null;
      currentPeriodUsage = {
        billingCycleStartMs: toMsString(u.billingCycleStart),
        billingCycleEndMs: toMsString(u.billingCycleEnd),
        enabled: Boolean(u.enabled),
        planUsage: pu
          ? {
              totalSpend: typeof pu.totalSpend === "number" ? pu.totalSpend : 0,
              includedSpend:
                typeof pu.includedSpend === "number" ? pu.includedSpend : 0,
              bonusSpend: typeof pu.bonusSpend === "number" ? pu.bonusSpend : 0,
              remaining: typeof pu.remaining === "number" ? pu.remaining : 0,
              limit: typeof pu.limit === "number" ? pu.limit : 0,
              autoPercentUsed:
                typeof pu.autoPercentUsed === "number" ? pu.autoPercentUsed : null,
              apiPercentUsed:
                typeof pu.apiPercentUsed === "number" ? pu.apiPercentUsed : null,
              totalPercentUsed:
                typeof pu.totalPercentUsed === "number" ? pu.totalPercentUsed : null,
            }
          : null,
      };
    } else if (usageRes.status === "rejected") {
      usageError = serializeError(usageRes.reason);
    }

    fs.writeSync(
      1,
      JSON.stringify({
        status: "authenticated",
        isAuthenticated: true,
        hasAccessToken: true,
        hasRefreshToken: true,
        hasApiKey: false,
        planInfo: planInfo,
        planError: planError,
        currentPeriodUsage: currentPeriodUsage,
        usageError: usageError,
      }) + "\n"
    );
    return 0;
  };
  wrapped.__gaebokchiWrapped = true;
  return wrapped;
}

Module._load = function (request, parent, isMain) {
  const loaded = origLoad.apply(this, arguments);
  if (
    loaded &&
    loaded.modules &&
    typeof loaded.modules["./src/commands/status.ts"] === "function" &&
    !loaded.modules["./src/commands/status.ts"].__gaebokchiWrapped
  ) {
    const origMod = loaded.modules["./src/commands/status.ts"];
    const wrappedMod = function (e, t, a) {
      const patchedA =
        a && typeof a.d === "function"
          ? new Proxy(a, {
              get(target, prop, receiver) {
                if (prop === "d") {
                  return function (exportsObj, defs) {
                    if (defs && typeof defs.handleStatus === "function") {
                      const origGetter = defs.handleStatus;
                      defs = Object.assign({}, defs, {
                        handleStatus: function () {
                          return wrapHandleStatus(origGetter());
                        },
                      });
                    }
                    return target.d(exportsObj, defs);
                  };
                }
                return Reflect.get(target, prop, receiver);
              },
            })
          : a;
      const ret = origMod.call(this, e, t, patchedA);
      if (
        t &&
        typeof t.handleStatus === "function" &&
        !t.handleStatus.__gaebokchiWrapped
      ) {
        t.handleStatus = wrapHandleStatus(t.handleStatus);
      }
      return ret;
    };
    wrappedMod.__gaebokchiWrapped = true;
    loaded.modules["./src/commands/status.ts"] = wrappedMod;
  }
  return loaded;
};
