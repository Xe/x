var htmx = (function () {
  "use strict";
  const Q = {
    onLoad: null,
    process: null,
    on: null,
    off: null,
    trigger: null,
    ajax: null,
    find: null,
    findAll: null,
    closest: null,
    values: function (e, t) {
      const n = cn(e, t || "post");
      return n.values;
    },
    remove: null,
    addClass: null,
    removeClass: null,
    toggleClass: null,
    takeClass: null,
    swap: null,
    defineExtension: null,
    removeExtension: null,
    logAll: null,
    logNone: null,
    logger: null,
    config: {
      historyEnabled: true,
      historyCacheSize: 10,
      refreshOnHistoryMiss: false,
      defaultSwapStyle: "innerHTML",
      defaultSwapDelay: 0,
      defaultSettleDelay: 20,
      includeIndicatorStyles: true,
      indicatorClass: "htmx-indicator",
      requestClass: "htmx-request",
      addedClass: "htmx-added",
      settlingClass: "htmx-settling",
      swappingClass: "htmx-swapping",
      allowEval: true,
      allowScriptTags: true,
      inlineScriptNonce: "",
      inlineStyleNonce: "",
      attributesToSettle: ["class", "style", "width", "height"],
      withCredentials: false,
      timeout: 0,
      wsReconnectDelay: "full-jitter",
      wsBinaryType: "blob",
      disableSelector: "[hx-disable], [data-hx-disable]",
      scrollBehavior: "instant",
      defaultFocusScroll: false,
      getCacheBusterParam: false,
      globalViewTransitions: false,
      methodsThatUseUrlParams: ["get", "delete"],
      selfRequestsOnly: true,
      ignoreTitle: false,
      scrollIntoViewOnBoost: true,
      triggerSpecsCache: null,
      disableInheritance: false,
      responseHandling: [
        { code: "204", swap: false },
        { code: "[23]..", swap: true },
        { code: "[45]..", swap: false, error: true },
      ],
      allowNestedOobSwaps: true,
    },
    parseInterval: null,
    _: null,
    version: "2.0.2",
  };
  Q.onLoad = $;
  Q.process = Dt;
  Q.on = be;
  Q.off = we;
  Q.trigger = de;
  Q.ajax = Hn;
  Q.find = r;
  Q.findAll = p;
  Q.closest = g;
  Q.remove = K;
  Q.addClass = Y;
  Q.removeClass = o;
  Q.toggleClass = W;
  Q.takeClass = ge;
  Q.swap = ze;
  Q.defineExtension = Bn;
  Q.removeExtension = Un;
  Q.logAll = z;
  Q.logNone = J;
  Q.parseInterval = h;
  Q._ = _;
  const n = {
    addTriggerHandler: Et,
    bodyContains: le,
    canAccessLocalStorage: j,
    findThisElement: Ee,
    filterValues: hn,
    swap: ze,
    hasAttribute: s,
    getAttributeValue: te,
    getClosestAttributeValue: re,
    getClosestMatch: T,
    getExpressionVars: Cn,
    getHeaders: dn,
    getInputValues: cn,
    getInternalData: ie,
    getSwapSpecification: pn,
    getTriggerSpecs: lt,
    getTarget: Ce,
    makeFragment: D,
    mergeObjects: ue,
    makeSettleInfo: xn,
    oobSwap: Te,
    querySelectorExt: ae,
    settleImmediately: Gt,
    shouldCancel: ht,
    triggerEvent: de,
    triggerErrorEvent: fe,
    withExtensions: Bt,
  };
  const v = ["get", "post", "put", "delete", "patch"];
  const O = v
    .map(function (e) {
      return "[hx-" + e + "], [data-hx-" + e + "]";
    })
    .join(", ");
  const R = e("head");
  function e(e, t = false) {
    return new RegExp(
      `<${e}(\\s[^>]*>|>)([\\s\\S]*?)<\\/${e}>`,
      t ? "gim" : "im",
    );
  }
  function h(e) {
    if (e == undefined) {
      return undefined;
    }
    let t = NaN;
    if (e.slice(-2) == "ms") {
      t = parseFloat(e.slice(0, -2));
    } else if (e.slice(-1) == "s") {
      t = parseFloat(e.slice(0, -1)) * 1e3;
    } else if (e.slice(-1) == "m") {
      t = parseFloat(e.slice(0, -1)) * 1e3 * 60;
    } else {
      t = parseFloat(e);
    }
    return isNaN(t) ? undefined : t;
  }
  function ee(e, t) {
    return e instanceof Element && e.getAttribute(t);
  }
  function s(e, t) {
    return (
      !!e.hasAttribute && (e.hasAttribute(t) || e.hasAttribute("data-" + t))
    );
  }
  function te(e, t) {
    return ee(e, t) || ee(e, "data-" + t);
  }
  function u(e) {
    const t = e.parentElement;
    if (!t && e.parentNode instanceof ShadowRoot) return e.parentNode;
    return t;
  }
  function ne() {
    return document;
  }
  function H(e, t) {
    return e.getRootNode ? e.getRootNode({ composed: t }) : ne();
  }
  function T(e, t) {
    while (e && !t(e)) {
      e = u(e);
    }
    return e || null;
  }
  function q(e, t, n) {
    const r = te(t, n);
    const o = te(t, "hx-disinherit");
    var i = te(t, "hx-inherit");
    if (e !== t) {
      if (Q.config.disableInheritance) {
        if (i && (i === "*" || i.split(" ").indexOf(n) >= 0)) {
          return r;
        } else {
          return null;
        }
      }
      if (o && (o === "*" || o.split(" ").indexOf(n) >= 0)) {
        return "unset";
      }
    }
    return r;
  }
  function re(t, n) {
    let r = null;
    T(t, function (e) {
      return !!(r = q(t, ce(e), n));
    });
    if (r !== "unset") {
      return r;
    }
  }
  function f(e, t) {
    const n =
      e instanceof Element &&
      (e.matches ||
        e.matchesSelector ||
        e.msMatchesSelector ||
        e.mozMatchesSelector ||
        e.webkitMatchesSelector ||
        e.oMatchesSelector);
    return !!n && n.call(e, t);
  }
  function L(e) {
    const t = /<([a-z][^\/\0>\x20\t\r\n\f]*)/i;
    const n = t.exec(e);
    if (n) {
      return n[1].toLowerCase();
    } else {
      return "";
    }
  }
  function N(e) {
    const t = new DOMParser();
    return t.parseFromString(e, "text/html");
  }
  function A(e, t) {
    while (t.childNodes.length > 0) {
      e.append(t.childNodes[0]);
    }
  }
  function I(e) {
    const t = ne().createElement("script");
    se(e.attributes, function (e) {
      t.setAttribute(e.name, e.value);
    });
    t.textContent = e.textContent;
    t.async = false;
    if (Q.config.inlineScriptNonce) {
      t.nonce = Q.config.inlineScriptNonce;
    }
    return t;
  }
  function P(e) {
    return (
      e.matches("script") &&
      (e.type === "text/javascript" || e.type === "module" || e.type === "")
    );
  }
  function k(e) {
    Array.from(e.querySelectorAll("script")).forEach((e) => {
      if (P(e)) {
        const t = I(e);
        const n = e.parentNode;
        try {
          n.insertBefore(t, e);
        } catch (e) {
          w(e);
        } finally {
          e.remove();
        }
      }
    });
  }
  function D(e) {
    const t = e.replace(R, "");
    const n = L(t);
    let r;
    if (n === "html") {
      r = new DocumentFragment();
      const i = N(e);
      A(r, i.body);
      r.title = i.title;
    } else if (n === "body") {
      r = new DocumentFragment();
      const i = N(t);
      A(r, i.body);
      r.title = i.title;
    } else {
      const i = N(
        '<body><template class="internal-htmx-wrapper">' +
          t +
          "</template></body>",
      );
      r = i.querySelector("template").content;
      r.title = i.title;
      var o = r.querySelector("title");
      if (o && o.parentNode === r) {
        o.remove();
        r.title = o.innerText;
      }
    }
    if (r) {
      if (Q.config.allowScriptTags) {
        k(r);
      } else {
        r.querySelectorAll("script").forEach((e) => e.remove());
      }
    }
    return r;
  }
  function oe(e) {
    if (e) {
      e();
    }
  }
  function t(e, t) {
    return Object.prototype.toString.call(e) === "[object " + t + "]";
  }
  function M(e) {
    return typeof e === "function";
  }
  function X(e) {
    return t(e, "Object");
  }
  function ie(e) {
    const t = "htmx-internal-data";
    let n = e[t];
    if (!n) {
      n = e[t] = {};
    }
    return n;
  }
  function F(t) {
    const n = [];
    if (t) {
      for (let e = 0; e < t.length; e++) {
        n.push(t[e]);
      }
    }
    return n;
  }
  function se(t, n) {
    if (t) {
      for (let e = 0; e < t.length; e++) {
        n(t[e]);
      }
    }
  }
  function B(e) {
    const t = e.getBoundingClientRect();
    const n = t.top;
    const r = t.bottom;
    return n < window.innerHeight && r >= 0;
  }
  function le(e) {
    const t = e.getRootNode && e.getRootNode();
    if (t && t instanceof window.ShadowRoot) {
      return ne().body.contains(t.host);
    } else {
      return ne().body.contains(e);
    }
  }
  function U(e) {
    return e.trim().split(/\s+/);
  }
  function ue(e, t) {
    for (const n in t) {
      if (t.hasOwnProperty(n)) {
        e[n] = t[n];
      }
    }
    return e;
  }
  function S(e) {
    try {
      return JSON.parse(e);
    } catch (e) {
      w(e);
      return null;
    }
  }
  function j() {
    const e = "htmx:localStorageTest";
    try {
      localStorage.setItem(e, e);
      localStorage.removeItem(e);
      return true;
    } catch (e) {
      return false;
    }
  }
  function V(t) {
    try {
      const e = new URL(t);
      if (e) {
        t = e.pathname + e.search;
      }
      if (!/^\/$/.test(t)) {
        t = t.replace(/\/+$/, "");
      }
      return t;
    } catch (e) {
      return t;
    }
  }
  function _(e) {
    return vn(ne().body, function () {
      return eval(e);
    });
  }
  function $(t) {
    const e = Q.on("htmx:load", function (e) {
      t(e.detail.elt);
    });
    return e;
  }
  function z() {
    Q.logger = function (e, t, n) {
      if (console) {
        console.log(t, e, n);
      }
    };
  }
  function J() {
    Q.logger = null;
  }
  function r(e, t) {
    if (typeof e !== "string") {
      return e.querySelector(t);
    } else {
      return r(ne(), e);
    }
  }
  function p(e, t) {
    if (typeof e !== "string") {
      return e.querySelectorAll(t);
    } else {
      return p(ne(), e);
    }
  }
  function E() {
    return window;
  }
  function K(e, t) {
    e = y(e);
    if (t) {
      E().setTimeout(function () {
        K(e);
        e = null;
      }, t);
    } else {
      u(e).removeChild(e);
    }
  }
  function ce(e) {
    return e instanceof Element ? e : null;
  }
  function G(e) {
    return e instanceof HTMLElement ? e : null;
  }
  function Z(e) {
    return typeof e === "string" ? e : null;
  }
  function d(e) {
    return e instanceof Element ||
      e instanceof Document ||
      e instanceof DocumentFragment
      ? e
      : null;
  }
  function Y(e, t, n) {
    e = ce(y(e));
    if (!e) {
      return;
    }
    if (n) {
      E().setTimeout(function () {
        Y(e, t);
        e = null;
      }, n);
    } else {
      e.classList && e.classList.add(t);
    }
  }
  function o(e, t, n) {
    let r = ce(y(e));
    if (!r) {
      return;
    }
    if (n) {
      E().setTimeout(function () {
        o(r, t);
        r = null;
      }, n);
    } else {
      if (r.classList) {
        r.classList.remove(t);
        if (r.classList.length === 0) {
          r.removeAttribute("class");
        }
      }
    }
  }
  function W(e, t) {
    e = y(e);
    e.classList.toggle(t);
  }
  function ge(e, t) {
    e = y(e);
    se(e.parentElement.children, function (e) {
      o(e, t);
    });
    Y(ce(e), t);
  }
  function g(e, t) {
    e = ce(y(e));
    if (e && e.closest) {
      return e.closest(t);
    } else {
      do {
        if (e == null || f(e, t)) {
          return e;
        }
      } while ((e = e && ce(u(e))));
      return null;
    }
  }
  function l(e, t) {
    return e.substring(0, t.length) === t;
  }
  function pe(e, t) {
    return e.substring(e.length - t.length) === t;
  }
  function i(e) {
    const t = e.trim();
    if (l(t, "<") && pe(t, "/>")) {
      return t.substring(1, t.length - 2);
    } else {
      return t;
    }
  }
  function m(e, t, n) {
    e = y(e);
    if (t.indexOf("closest ") === 0) {
      return [g(ce(e), i(t.substr(8)))];
    } else if (t.indexOf("find ") === 0) {
      return [r(d(e), i(t.substr(5)))];
    } else if (t === "next") {
      return [ce(e).nextElementSibling];
    } else if (t.indexOf("next ") === 0) {
      return [me(e, i(t.substr(5)), !!n)];
    } else if (t === "previous") {
      return [ce(e).previousElementSibling];
    } else if (t.indexOf("previous ") === 0) {
      return [ye(e, i(t.substr(9)), !!n)];
    } else if (t === "document") {
      return [document];
    } else if (t === "window") {
      return [window];
    } else if (t === "body") {
      return [document.body];
    } else if (t === "root") {
      return [H(e, !!n)];
    } else if (t.indexOf("global ") === 0) {
      return m(e, t.slice(7), true);
    } else {
      return F(d(H(e, !!n)).querySelectorAll(i(t)));
    }
  }
  var me = function (t, e, n) {
    const r = d(H(t, n)).querySelectorAll(e);
    for (let e = 0; e < r.length; e++) {
      const o = r[e];
      if (o.compareDocumentPosition(t) === Node.DOCUMENT_POSITION_PRECEDING) {
        return o;
      }
    }
  };
  var ye = function (t, e, n) {
    const r = d(H(t, n)).querySelectorAll(e);
    for (let e = r.length - 1; e >= 0; e--) {
      const o = r[e];
      if (o.compareDocumentPosition(t) === Node.DOCUMENT_POSITION_FOLLOWING) {
        return o;
      }
    }
  };
  function ae(e, t) {
    if (typeof e !== "string") {
      return m(e, t)[0];
    } else {
      return m(ne().body, e)[0];
    }
  }
  function y(e, t) {
    if (typeof e === "string") {
      return r(d(t) || document, e);
    } else {
      return e;
    }
  }
  function xe(e, t, n) {
    if (M(t)) {
      return { target: ne().body, event: Z(e), listener: t };
    } else {
      return { target: y(e), event: Z(t), listener: n };
    }
  }
  function be(t, n, r) {
    _n(function () {
      const e = xe(t, n, r);
      e.target.addEventListener(e.event, e.listener);
    });
    const e = M(n);
    return e ? n : r;
  }
  function we(t, n, r) {
    _n(function () {
      const e = xe(t, n, r);
      e.target.removeEventListener(e.event, e.listener);
    });
    return M(n) ? n : r;
  }
  const ve = ne().createElement("output");
  function Se(e, t) {
    const n = re(e, t);
    if (n) {
      if (n === "this") {
        return [Ee(e, t)];
      } else {
        const r = m(e, n);
        if (r.length === 0) {
          w('The selector "' + n + '" on ' + t + " returned no matches!");
          return [ve];
        } else {
          return r;
        }
      }
    }
  }
  function Ee(e, t) {
    return ce(
      T(e, function (e) {
        return te(ce(e), t) != null;
      }),
    );
  }
  function Ce(e) {
    const t = re(e, "hx-target");
    if (t) {
      if (t === "this") {
        return Ee(e, "hx-target");
      } else {
        return ae(e, t);
      }
    } else {
      const n = ie(e);
      if (n.boosted) {
        return ne().body;
      } else {
        return e;
      }
    }
  }
  function Oe(t) {
    const n = Q.config.attributesToSettle;
    for (let e = 0; e < n.length; e++) {
      if (t === n[e]) {
        return true;
      }
    }
    return false;
  }
  function Re(t, n) {
    se(t.attributes, function (e) {
      if (!n.hasAttribute(e.name) && Oe(e.name)) {
        t.removeAttribute(e.name);
      }
    });
    se(n.attributes, function (e) {
      if (Oe(e.name)) {
        t.setAttribute(e.name, e.value);
      }
    });
  }
  function He(t, e) {
    const n = jn(e);
    for (let e = 0; e < n.length; e++) {
      const r = n[e];
      try {
        if (r.isInlineSwap(t)) {
          return true;
        }
      } catch (e) {
        w(e);
      }
    }
    return t === "outerHTML";
  }
  function Te(e, o, i) {
    let t = "#" + ee(o, "id");
    let s = "outerHTML";
    if (e === "true") {
    } else if (e.indexOf(":") > 0) {
      s = e.substr(0, e.indexOf(":"));
      t = e.substr(e.indexOf(":") + 1, e.length);
    } else {
      s = e;
    }
    const n = ne().querySelectorAll(t);
    if (n) {
      se(n, function (e) {
        let t;
        const n = o.cloneNode(true);
        t = ne().createDocumentFragment();
        t.appendChild(n);
        if (!He(s, e)) {
          t = d(n);
        }
        const r = { shouldSwap: true, target: e, fragment: t };
        if (!de(e, "htmx:oobBeforeSwap", r)) return;
        e = r.target;
        if (r.shouldSwap) {
          _e(s, e, e, t, i);
        }
        se(i.elts, function (e) {
          de(e, "htmx:oobAfterSwap", r);
        });
      });
      o.parentNode.removeChild(o);
    } else {
      o.parentNode.removeChild(o);
      fe(ne().body, "htmx:oobErrorNoTarget", { content: o });
    }
    return e;
  }
  function qe(e) {
    se(p(e, "[hx-preserve], [data-hx-preserve]"), function (e) {
      const t = te(e, "id");
      const n = ne().getElementById(t);
      if (n != null) {
        e.parentNode.replaceChild(n, e);
      }
    });
  }
  function Le(l, e, u) {
    se(e.querySelectorAll("[id]"), function (t) {
      const n = ee(t, "id");
      if (n && n.length > 0) {
        const r = n.replace("'", "\\'");
        const o = t.tagName.replace(":", "\\:");
        const e = d(l);
        const i = e && e.querySelector(o + "[id='" + r + "']");
        if (i && i !== e) {
          const s = t.cloneNode();
          Re(t, i);
          u.tasks.push(function () {
            Re(t, s);
          });
        }
      }
    });
  }
  function Ne(e) {
    return function () {
      o(e, Q.config.addedClass);
      Dt(ce(e));
      Ae(d(e));
      de(e, "htmx:load");
    };
  }
  function Ae(e) {
    const t = "[autofocus]";
    const n = G(f(e, t) ? e : e.querySelector(t));
    if (n != null) {
      n.focus();
    }
  }
  function c(e, t, n, r) {
    Le(e, n, r);
    while (n.childNodes.length > 0) {
      const o = n.firstChild;
      Y(ce(o), Q.config.addedClass);
      e.insertBefore(o, t);
      if (o.nodeType !== Node.TEXT_NODE && o.nodeType !== Node.COMMENT_NODE) {
        r.tasks.push(Ne(o));
      }
    }
  }
  function Ie(e, t) {
    let n = 0;
    while (n < e.length) {
      t = ((t << 5) - t + e.charCodeAt(n++)) | 0;
    }
    return t;
  }
  function Pe(t) {
    let n = 0;
    if (t.attributes) {
      for (let e = 0; e < t.attributes.length; e++) {
        const r = t.attributes[e];
        if (r.value) {
          n = Ie(r.name, n);
          n = Ie(r.value, n);
        }
      }
    }
    return n;
  }
  function ke(t) {
    const n = ie(t);
    if (n.onHandlers) {
      for (let e = 0; e < n.onHandlers.length; e++) {
        const r = n.onHandlers[e];
        we(t, r.event, r.listener);
      }
      delete n.onHandlers;
    }
  }
  function De(e) {
    const t = ie(e);
    if (t.timeout) {
      clearTimeout(t.timeout);
    }
    if (t.listenerInfos) {
      se(t.listenerInfos, function (e) {
        if (e.on) {
          we(e.on, e.trigger, e.listener);
        }
      });
    }
    ke(e);
    se(Object.keys(t), function (e) {
      delete t[e];
    });
  }
  function a(e) {
    de(e, "htmx:beforeCleanupElement");
    De(e);
    if (e.children) {
      se(e.children, function (e) {
        a(e);
      });
    }
  }
  function Me(t, e, n) {
    if (t instanceof Element && t.tagName === "BODY") {
      return Ve(t, e, n);
    }
    let r;
    const o = t.previousSibling;
    c(u(t), t, e, n);
    if (o == null) {
      r = u(t).firstChild;
    } else {
      r = o.nextSibling;
    }
    n.elts = n.elts.filter(function (e) {
      return e !== t;
    });
    while (r && r !== t) {
      if (r instanceof Element) {
        n.elts.push(r);
      }
      r = r.nextSibling;
    }
    a(t);
    if (t instanceof Element) {
      t.remove();
    } else {
      t.parentNode.removeChild(t);
    }
  }
  function Xe(e, t, n) {
    return c(e, e.firstChild, t, n);
  }
  function Fe(e, t, n) {
    return c(u(e), e, t, n);
  }
  function Be(e, t, n) {
    return c(e, null, t, n);
  }
  function Ue(e, t, n) {
    return c(u(e), e.nextSibling, t, n);
  }
  function je(e) {
    a(e);
    return u(e).removeChild(e);
  }
  function Ve(e, t, n) {
    const r = e.firstChild;
    c(e, r, t, n);
    if (r) {
      while (r.nextSibling) {
        a(r.nextSibling);
        e.removeChild(r.nextSibling);
      }
      a(r);
      e.removeChild(r);
    }
  }
  function _e(t, e, n, r, o) {
    switch (t) {
      case "none":
        return;
      case "outerHTML":
        Me(n, r, o);
        return;
      case "afterbegin":
        Xe(n, r, o);
        return;
      case "beforebegin":
        Fe(n, r, o);
        return;
      case "beforeend":
        Be(n, r, o);
        return;
      case "afterend":
        Ue(n, r, o);
        return;
      case "delete":
        je(n);
        return;
      default:
        var i = jn(e);
        for (let e = 0; e < i.length; e++) {
          const s = i[e];
          try {
            const l = s.handleSwap(t, n, r, o);
            if (l) {
              if (Array.isArray(l)) {
                for (let e = 0; e < l.length; e++) {
                  const u = l[e];
                  if (
                    u.nodeType !== Node.TEXT_NODE &&
                    u.nodeType !== Node.COMMENT_NODE
                  ) {
                    o.tasks.push(Ne(u));
                  }
                }
              }
              return;
            }
          } catch (e) {
            w(e);
          }
        }
        if (t === "innerHTML") {
          Ve(n, r, o);
        } else {
          _e(Q.config.defaultSwapStyle, e, n, r, o);
        }
    }
  }
  function $e(e, n) {
    var t = p(e, "[hx-swap-oob], [data-hx-swap-oob]");
    se(t, function (e) {
      if (Q.config.allowNestedOobSwaps || e.parentElement === null) {
        const t = te(e, "hx-swap-oob");
        if (t != null) {
          Te(t, e, n);
        }
      } else {
        e.removeAttribute("hx-swap-oob");
        e.removeAttribute("data-hx-swap-oob");
      }
    });
    return t.length > 0;
  }
  function ze(e, t, r, o) {
    if (!o) {
      o = {};
    }
    e = y(e);
    const n = document.activeElement;
    let i = {};
    try {
      i = {
        elt: n,
        start: n ? n.selectionStart : null,
        end: n ? n.selectionEnd : null,
      };
    } catch (e) {}
    const s = xn(e);
    if (r.swapStyle === "textContent") {
      e.textContent = t;
    } else {
      let n = D(t);
      s.title = n.title;
      if (o.selectOOB) {
        const u = o.selectOOB.split(",");
        for (let t = 0; t < u.length; t++) {
          const c = u[t].split(":", 2);
          let e = c[0].trim();
          if (e.indexOf("#") === 0) {
            e = e.substring(1);
          }
          const a = c[1] || "true";
          const f = n.querySelector("#" + e);
          if (f) {
            Te(a, f, s);
          }
        }
      }
      $e(n, s);
      se(p(n, "template"), function (e) {
        if ($e(e.content, s)) {
          e.remove();
        }
      });
      if (o.select) {
        const d = ne().createDocumentFragment();
        se(n.querySelectorAll(o.select), function (e) {
          d.appendChild(e);
        });
        n = d;
      }
      qe(n);
      _e(r.swapStyle, o.contextElement, e, n, s);
    }
    if (i.elt && !le(i.elt) && ee(i.elt, "id")) {
      const h = document.getElementById(ee(i.elt, "id"));
      const g = {
        preventScroll:
          r.focusScroll !== undefined
            ? !r.focusScroll
            : !Q.config.defaultFocusScroll,
      };
      if (h) {
        if (i.start && h.setSelectionRange) {
          try {
            h.setSelectionRange(i.start, i.end);
          } catch (e) {}
        }
        h.focus(g);
      }
    }
    e.classList.remove(Q.config.swappingClass);
    se(s.elts, function (e) {
      if (e.classList) {
        e.classList.add(Q.config.settlingClass);
      }
      de(e, "htmx:afterSwap", o.eventInfo);
    });
    if (o.afterSwapCallback) {
      o.afterSwapCallback();
    }
    if (!r.ignoreTitle) {
      Dn(s.title);
    }
    const l = function () {
      se(s.tasks, function (e) {
        e.call();
      });
      se(s.elts, function (e) {
        if (e.classList) {
          e.classList.remove(Q.config.settlingClass);
        }
        de(e, "htmx:afterSettle", o.eventInfo);
      });
      if (o.anchor) {
        const e = ce(y("#" + o.anchor));
        if (e) {
          e.scrollIntoView({ block: "start", behavior: "auto" });
        }
      }
      bn(s.elts, r);
      if (o.afterSettleCallback) {
        o.afterSettleCallback();
      }
    };
    if (r.settleDelay > 0) {
      E().setTimeout(l, r.settleDelay);
    } else {
      l();
    }
  }
  function Je(e, t, n) {
    const r = e.getResponseHeader(t);
    if (r.indexOf("{") === 0) {
      const o = S(r);
      for (const i in o) {
        if (o.hasOwnProperty(i)) {
          let e = o[i];
          if (X(e)) {
            n = e.target !== undefined ? e.target : n;
          } else {
            e = { value: e };
          }
          de(n, i, e);
        }
      }
    } else {
      const s = r.split(",");
      for (let e = 0; e < s.length; e++) {
        de(n, s[e].trim(), []);
      }
    }
  }
  const Ke = /\s/;
  const x = /[\s,]/;
  const Ge = /[_$a-zA-Z]/;
  const Ze = /[_$a-zA-Z0-9]/;
  const Ye = ['"', "'", "/"];
  const We = /[^\s]/;
  const Qe = /[{(]/;
  const et = /[})]/;
  function tt(e) {
    const t = [];
    let n = 0;
    while (n < e.length) {
      if (Ge.exec(e.charAt(n))) {
        var r = n;
        while (Ze.exec(e.charAt(n + 1))) {
          n++;
        }
        t.push(e.substr(r, n - r + 1));
      } else if (Ye.indexOf(e.charAt(n)) !== -1) {
        const o = e.charAt(n);
        var r = n;
        n++;
        while (n < e.length && e.charAt(n) !== o) {
          if (e.charAt(n) === "\\") {
            n++;
          }
          n++;
        }
        t.push(e.substr(r, n - r + 1));
      } else {
        const i = e.charAt(n);
        t.push(i);
      }
      n++;
    }
    return t;
  }
  function nt(e, t, n) {
    return (
      Ge.exec(e.charAt(0)) &&
      e !== "true" &&
      e !== "false" &&
      e !== "this" &&
      e !== n &&
      t !== "."
    );
  }
  function rt(r, o, i) {
    if (o[0] === "[") {
      o.shift();
      let e = 1;
      let t = " return (function(" + i + "){ return (";
      let n = null;
      while (o.length > 0) {
        const s = o[0];
        if (s === "]") {
          e--;
          if (e === 0) {
            if (n === null) {
              t = t + "true";
            }
            o.shift();
            t += ")})";
            try {
              const l = vn(
                r,
                function () {
                  return Function(t)();
                },
                function () {
                  return true;
                },
              );
              l.source = t;
              return l;
            } catch (e) {
              fe(ne().body, "htmx:syntax:error", { error: e, source: t });
              return null;
            }
          }
        } else if (s === "[") {
          e++;
        }
        if (nt(s, n, i)) {
          t +=
            "((" +
            i +
            "." +
            s +
            ") ? (" +
            i +
            "." +
            s +
            ") : (window." +
            s +
            "))";
        } else {
          t = t + s;
        }
        n = o.shift();
      }
    }
  }
  function b(e, t) {
    let n = "";
    while (e.length > 0 && !t.test(e[0])) {
      n += e.shift();
    }
    return n;
  }
  function ot(e) {
    let t;
    if (e.length > 0 && Qe.test(e[0])) {
      e.shift();
      t = b(e, et).trim();
      e.shift();
    } else {
      t = b(e, x);
    }
    return t;
  }
  const it = "input, textarea, select";
  function st(e, t, n) {
    const r = [];
    const o = tt(t);
    do {
      b(o, We);
      const l = o.length;
      const u = b(o, /[,\[\s]/);
      if (u !== "") {
        if (u === "every") {
          const c = { trigger: "every" };
          b(o, We);
          c.pollInterval = h(b(o, /[,\[\s]/));
          b(o, We);
          var i = rt(e, o, "event");
          if (i) {
            c.eventFilter = i;
          }
          r.push(c);
        } else {
          const a = { trigger: u };
          var i = rt(e, o, "event");
          if (i) {
            a.eventFilter = i;
          }
          while (o.length > 0 && o[0] !== ",") {
            b(o, We);
            const f = o.shift();
            if (f === "changed") {
              a.changed = true;
            } else if (f === "once") {
              a.once = true;
            } else if (f === "consume") {
              a.consume = true;
            } else if (f === "delay" && o[0] === ":") {
              o.shift();
              a.delay = h(b(o, x));
            } else if (f === "from" && o[0] === ":") {
              o.shift();
              if (Qe.test(o[0])) {
                var s = ot(o);
              } else {
                var s = b(o, x);
                if (
                  s === "closest" ||
                  s === "find" ||
                  s === "next" ||
                  s === "previous"
                ) {
                  o.shift();
                  const d = ot(o);
                  if (d.length > 0) {
                    s += " " + d;
                  }
                }
              }
              a.from = s;
            } else if (f === "target" && o[0] === ":") {
              o.shift();
              a.target = ot(o);
            } else if (f === "throttle" && o[0] === ":") {
              o.shift();
              a.throttle = h(b(o, x));
            } else if (f === "queue" && o[0] === ":") {
              o.shift();
              a.queue = b(o, x);
            } else if (f === "root" && o[0] === ":") {
              o.shift();
              a[f] = ot(o);
            } else if (f === "threshold" && o[0] === ":") {
              o.shift();
              a[f] = b(o, x);
            } else {
              fe(e, "htmx:syntax:error", { token: o.shift() });
            }
          }
          r.push(a);
        }
      }
      if (o.length === l) {
        fe(e, "htmx:syntax:error", { token: o.shift() });
      }
      b(o, We);
    } while (o[0] === "," && o.shift());
    if (n) {
      n[t] = r;
    }
    return r;
  }
  function lt(e) {
    const t = te(e, "hx-trigger");
    let n = [];
    if (t) {
      const r = Q.config.triggerSpecsCache;
      n = (r && r[t]) || st(e, t, r);
    }
    if (n.length > 0) {
      return n;
    } else if (f(e, "form")) {
      return [{ trigger: "submit" }];
    } else if (f(e, 'input[type="button"], input[type="submit"]')) {
      return [{ trigger: "click" }];
    } else if (f(e, it)) {
      return [{ trigger: "change" }];
    } else {
      return [{ trigger: "click" }];
    }
  }
  function ut(e) {
    ie(e).cancelled = true;
  }
  function ct(e, t, n) {
    const r = ie(e);
    r.timeout = E().setTimeout(function () {
      if (le(e) && r.cancelled !== true) {
        if (!pt(n, e, Xt("hx:poll:trigger", { triggerSpec: n, target: e }))) {
          t(e);
        }
        ct(e, t, n);
      }
    }, n.pollInterval);
  }
  function at(e) {
    return (
      location.hostname === e.hostname &&
      ee(e, "href") &&
      ee(e, "href").indexOf("#") !== 0
    );
  }
  function ft(e) {
    return g(e, Q.config.disableSelector);
  }
  function dt(t, n, e) {
    if (
      (t instanceof HTMLAnchorElement &&
        at(t) &&
        (t.target === "" || t.target === "_self")) ||
      (t.tagName === "FORM" &&
        String(ee(t, "method")).toLowerCase() !== "dialog")
    ) {
      n.boosted = true;
      let r, o;
      if (t.tagName === "A") {
        r = "get";
        o = ee(t, "href");
      } else {
        const i = ee(t, "method");
        r = i ? i.toLowerCase() : "get";
        if (r === "get") {
        }
        o = ee(t, "action");
      }
      e.forEach(function (e) {
        mt(
          t,
          function (e, t) {
            const n = ce(e);
            if (ft(n)) {
              a(n);
              return;
            }
            he(r, o, n, t);
          },
          n,
          e,
          true,
        );
      });
    }
  }
  function ht(e, t) {
    const n = ce(t);
    if (!n) {
      return false;
    }
    if (e.type === "submit" || e.type === "click") {
      if (n.tagName === "FORM") {
        return true;
      }
      if (f(n, 'input[type="submit"], button') && g(n, "form") !== null) {
        return true;
      }
      if (
        n instanceof HTMLAnchorElement &&
        n.href &&
        (n.getAttribute("href") === "#" ||
          n.getAttribute("href").indexOf("#") !== 0)
      ) {
        return true;
      }
    }
    return false;
  }
  function gt(e, t) {
    return (
      ie(e).boosted &&
      e instanceof HTMLAnchorElement &&
      t.type === "click" &&
      (t.ctrlKey || t.metaKey)
    );
  }
  function pt(e, t, n) {
    const r = e.eventFilter;
    if (r) {
      try {
        return r.call(t, n) !== true;
      } catch (e) {
        const o = r.source;
        fe(ne().body, "htmx:eventFilter:error", { error: e, source: o });
        return true;
      }
    }
    return false;
  }
  function mt(s, l, e, u, c) {
    const a = ie(s);
    let t;
    if (u.from) {
      t = m(s, u.from);
    } else {
      t = [s];
    }
    if (u.changed) {
      t.forEach(function (e) {
        const t = ie(e);
        t.lastValue = e.value;
      });
    }
    se(t, function (o) {
      const i = function (e) {
        if (!le(s)) {
          o.removeEventListener(u.trigger, i);
          return;
        }
        if (gt(s, e)) {
          return;
        }
        if (c || ht(e, s)) {
          e.preventDefault();
        }
        if (pt(u, s, e)) {
          return;
        }
        const t = ie(e);
        t.triggerSpec = u;
        if (t.handledFor == null) {
          t.handledFor = [];
        }
        if (t.handledFor.indexOf(s) < 0) {
          t.handledFor.push(s);
          if (u.consume) {
            e.stopPropagation();
          }
          if (u.target && e.target) {
            if (!f(ce(e.target), u.target)) {
              return;
            }
          }
          if (u.once) {
            if (a.triggeredOnce) {
              return;
            } else {
              a.triggeredOnce = true;
            }
          }
          if (u.changed) {
            const n = ie(o);
            const r = o.value;
            if (n.lastValue === r) {
              return;
            }
            n.lastValue = r;
          }
          if (a.delayed) {
            clearTimeout(a.delayed);
          }
          if (a.throttle) {
            return;
          }
          if (u.throttle > 0) {
            if (!a.throttle) {
              de(s, "htmx:trigger");
              l(s, e);
              a.throttle = E().setTimeout(function () {
                a.throttle = null;
              }, u.throttle);
            }
          } else if (u.delay > 0) {
            a.delayed = E().setTimeout(function () {
              de(s, "htmx:trigger");
              l(s, e);
            }, u.delay);
          } else {
            de(s, "htmx:trigger");
            l(s, e);
          }
        }
      };
      if (e.listenerInfos == null) {
        e.listenerInfos = [];
      }
      e.listenerInfos.push({ trigger: u.trigger, listener: i, on: o });
      o.addEventListener(u.trigger, i);
    });
  }
  let yt = false;
  let xt = null;
  function bt() {
    if (!xt) {
      xt = function () {
        yt = true;
      };
      window.addEventListener("scroll", xt);
      setInterval(function () {
        if (yt) {
          yt = false;
          se(
            ne().querySelectorAll(
              "[hx-trigger*='revealed'],[data-hx-trigger*='revealed']",
            ),
            function (e) {
              wt(e);
            },
          );
        }
      }, 200);
    }
  }
  function wt(e) {
    if (!s(e, "data-hx-revealed") && B(e)) {
      e.setAttribute("data-hx-revealed", "true");
      const t = ie(e);
      if (t.initHash) {
        de(e, "revealed");
      } else {
        e.addEventListener(
          "htmx:afterProcessNode",
          function () {
            de(e, "revealed");
          },
          { once: true },
        );
      }
    }
  }
  function vt(e, t, n, r) {
    const o = function () {
      if (!n.loaded) {
        n.loaded = true;
        t(e);
      }
    };
    if (r > 0) {
      E().setTimeout(o, r);
    } else {
      o();
    }
  }
  function St(t, n, e) {
    let i = false;
    se(v, function (r) {
      if (s(t, "hx-" + r)) {
        const o = te(t, "hx-" + r);
        i = true;
        n.path = o;
        n.verb = r;
        e.forEach(function (e) {
          Et(t, e, n, function (e, t) {
            const n = ce(e);
            if (g(n, Q.config.disableSelector)) {
              a(n);
              return;
            }
            he(r, o, n, t);
          });
        });
      }
    });
    return i;
  }
  function Et(r, e, t, n) {
    if (e.trigger === "revealed") {
      bt();
      mt(r, n, t, e);
      wt(ce(r));
    } else if (e.trigger === "intersect") {
      const o = {};
      if (e.root) {
        o.root = ae(r, e.root);
      }
      if (e.threshold) {
        o.threshold = parseFloat(e.threshold);
      }
      const i = new IntersectionObserver(function (t) {
        for (let e = 0; e < t.length; e++) {
          const n = t[e];
          if (n.isIntersecting) {
            de(r, "intersect");
            break;
          }
        }
      }, o);
      i.observe(ce(r));
      mt(ce(r), n, t, e);
    } else if (e.trigger === "load") {
      if (!pt(e, r, Xt("load", { elt: r }))) {
        vt(ce(r), n, t, e.delay);
      }
    } else if (e.pollInterval > 0) {
      t.polling = true;
      ct(ce(r), n, e);
    } else {
      mt(r, n, t, e);
    }
  }
  function Ct(e) {
    const t = ce(e);
    if (!t) {
      return false;
    }
    const n = t.attributes;
    for (let e = 0; e < n.length; e++) {
      const r = n[e].name;
      if (
        l(r, "hx-on:") ||
        l(r, "data-hx-on:") ||
        l(r, "hx-on-") ||
        l(r, "data-hx-on-")
      ) {
        return true;
      }
    }
    return false;
  }
  const Ot = new XPathEvaluator().createExpression(
    './/*[@*[ starts-with(name(), "hx-on:") or starts-with(name(), "data-hx-on:") or' +
      ' starts-with(name(), "hx-on-") or starts-with(name(), "data-hx-on-") ]]',
  );
  function Rt(e, t) {
    if (Ct(e)) {
      t.push(ce(e));
    }
    const n = Ot.evaluate(e);
    let r = null;
    while ((r = n.iterateNext())) t.push(ce(r));
  }
  function Ht(e) {
    const t = [];
    if (e instanceof DocumentFragment) {
      for (const n of e.childNodes) {
        Rt(n, t);
      }
    } else {
      Rt(e, t);
    }
    return t;
  }
  function Tt(e) {
    if (e.querySelectorAll) {
      const n =
        ", [hx-boost] a, [data-hx-boost] a, a[hx-boost], a[data-hx-boost]";
      const r = [];
      for (const i in Xn) {
        const s = Xn[i];
        if (s.getSelectors) {
          var t = s.getSelectors();
          if (t) {
            r.push(t);
          }
        }
      }
      const o = e.querySelectorAll(
        O +
          n +
          ", form, [type='submit']," +
          " [hx-ext], [data-hx-ext], [hx-trigger], [data-hx-trigger]" +
          r
            .flat()
            .map((e) => ", " + e)
            .join(""),
      );
      return o;
    } else {
      return [];
    }
  }
  function qt(e) {
    const t = g(ce(e.target), "button, input[type='submit']");
    const n = Nt(e);
    if (n) {
      n.lastButtonClicked = t;
    }
  }
  function Lt(e) {
    const t = Nt(e);
    if (t) {
      t.lastButtonClicked = null;
    }
  }
  function Nt(e) {
    const t = g(ce(e.target), "button, input[type='submit']");
    if (!t) {
      return;
    }
    const n = y("#" + ee(t, "form"), t.getRootNode()) || g(t, "form");
    if (!n) {
      return;
    }
    return ie(n);
  }
  function At(e) {
    e.addEventListener("click", qt);
    e.addEventListener("focusin", qt);
    e.addEventListener("focusout", Lt);
  }
  function It(t, e, n) {
    const r = ie(t);
    if (!Array.isArray(r.onHandlers)) {
      r.onHandlers = [];
    }
    let o;
    const i = function (e) {
      vn(t, function () {
        if (ft(t)) {
          return;
        }
        if (!o) {
          o = new Function("event", n);
        }
        o.call(t, e);
      });
    };
    t.addEventListener(e, i);
    r.onHandlers.push({ event: e, listener: i });
  }
  function Pt(t) {
    ke(t);
    for (let e = 0; e < t.attributes.length; e++) {
      const n = t.attributes[e].name;
      const r = t.attributes[e].value;
      if (l(n, "hx-on") || l(n, "data-hx-on")) {
        const o = n.indexOf("-on") + 3;
        const i = n.slice(o, o + 1);
        if (i === "-" || i === ":") {
          let e = n.slice(o + 1);
          if (l(e, ":")) {
            e = "htmx" + e;
          } else if (l(e, "-")) {
            e = "htmx:" + e.slice(1);
          } else if (l(e, "htmx-")) {
            e = "htmx:" + e.slice(5);
          }
          It(t, e, r);
        }
      }
    }
  }
  function kt(t) {
    if (g(t, Q.config.disableSelector)) {
      a(t);
      return;
    }
    const n = ie(t);
    if (n.initHash !== Pe(t)) {
      De(t);
      n.initHash = Pe(t);
      de(t, "htmx:beforeProcessNode");
      if (t.value) {
        n.lastValue = t.value;
      }
      const e = lt(t);
      const r = St(t, n, e);
      if (!r) {
        if (re(t, "hx-boost") === "true") {
          dt(t, n, e);
        } else if (s(t, "hx-trigger")) {
          e.forEach(function (e) {
            Et(t, e, n, function () {});
          });
        }
      }
      if (
        t.tagName === "FORM" ||
        (ee(t, "type") === "submit" && s(t, "form"))
      ) {
        At(t);
      }
      de(t, "htmx:afterProcessNode");
    }
  }
  function Dt(e) {
    e = y(e);
    if (g(e, Q.config.disableSelector)) {
      a(e);
      return;
    }
    kt(e);
    se(Tt(e), function (e) {
      kt(e);
    });
    se(Ht(e), Pt);
  }
  function Mt(e) {
    return e.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();
  }
  function Xt(e, t) {
    let n;
    if (window.CustomEvent && typeof window.CustomEvent === "function") {
      n = new CustomEvent(e, {
        bubbles: true,
        cancelable: true,
        composed: true,
        detail: t,
      });
    } else {
      n = ne().createEvent("CustomEvent");
      n.initCustomEvent(e, true, true, t);
    }
    return n;
  }
  function fe(e, t, n) {
    de(e, t, ue({ error: t }, n));
  }
  function Ft(e) {
    return e === "htmx:afterProcessNode";
  }
  function Bt(e, t) {
    se(jn(e), function (e) {
      try {
        t(e);
      } catch (e) {
        w(e);
      }
    });
  }
  function w(e) {
    if (console.error) {
      console.error(e);
    } else if (console.log) {
      console.log("ERROR: ", e);
    }
  }
  function de(e, t, n) {
    e = y(e);
    if (n == null) {
      n = {};
    }
    n.elt = e;
    const r = Xt(t, n);
    if (Q.logger && !Ft(t)) {
      Q.logger(e, t, n);
    }
    if (n.error) {
      w(n.error);
      de(e, "htmx:error", { errorInfo: n });
    }
    let o = e.dispatchEvent(r);
    const i = Mt(t);
    if (o && i !== t) {
      const s = Xt(i, r.detail);
      o = o && e.dispatchEvent(s);
    }
    Bt(ce(e), function (e) {
      o = o && e.onEvent(t, r) !== false && !r.defaultPrevented;
    });
    return o;
  }
  let Ut = location.pathname + location.search;
  function jt() {
    const e = ne().querySelector("[hx-history-elt],[data-hx-history-elt]");
    return e || ne().body;
  }
  function Vt(t, e) {
    if (!j()) {
      return;
    }
    const n = $t(e);
    const r = ne().title;
    const o = window.scrollY;
    if (Q.config.historyCacheSize <= 0) {
      localStorage.removeItem("htmx-history-cache");
      return;
    }
    t = V(t);
    const i = S(localStorage.getItem("htmx-history-cache")) || [];
    for (let e = 0; e < i.length; e++) {
      if (i[e].url === t) {
        i.splice(e, 1);
        break;
      }
    }
    const s = { url: t, content: n, title: r, scroll: o };
    de(ne().body, "htmx:historyItemCreated", { item: s, cache: i });
    i.push(s);
    while (i.length > Q.config.historyCacheSize) {
      i.shift();
    }
    while (i.length > 0) {
      try {
        localStorage.setItem("htmx-history-cache", JSON.stringify(i));
        break;
      } catch (e) {
        fe(ne().body, "htmx:historyCacheError", { cause: e, cache: i });
        i.shift();
      }
    }
  }
  function _t(t) {
    if (!j()) {
      return null;
    }
    t = V(t);
    const n = S(localStorage.getItem("htmx-history-cache")) || [];
    for (let e = 0; e < n.length; e++) {
      if (n[e].url === t) {
        return n[e];
      }
    }
    return null;
  }
  function $t(e) {
    const t = Q.config.requestClass;
    const n = e.cloneNode(true);
    se(p(n, "." + t), function (e) {
      o(e, t);
    });
    se(p(n, "[data-disabled-by-htmx]"), function (e) {
      e.removeAttribute("disabled");
    });
    return n.innerHTML;
  }
  function zt() {
    const e = jt();
    const t = Ut || location.pathname + location.search;
    let n;
    try {
      n = ne().querySelector(
        '[hx-history="false" i],[data-hx-history="false" i]',
      );
    } catch (e) {
      n = ne().querySelector('[hx-history="false"],[data-hx-history="false"]');
    }
    if (!n) {
      de(ne().body, "htmx:beforeHistorySave", { path: t, historyElt: e });
      Vt(t, e);
    }
    if (Q.config.historyEnabled)
      history.replaceState({ htmx: true }, ne().title, window.location.href);
  }
  function Jt(e) {
    if (Q.config.getCacheBusterParam) {
      e = e.replace(/org\.htmx\.cache-buster=[^&]*&?/, "");
      if (pe(e, "&") || pe(e, "?")) {
        e = e.slice(0, -1);
      }
    }
    if (Q.config.historyEnabled) {
      history.pushState({ htmx: true }, "", e);
    }
    Ut = e;
  }
  function Kt(e) {
    if (Q.config.historyEnabled) history.replaceState({ htmx: true }, "", e);
    Ut = e;
  }
  function Gt(e) {
    se(e, function (e) {
      e.call(undefined);
    });
  }
  function Zt(o) {
    const e = new XMLHttpRequest();
    const i = { path: o, xhr: e };
    de(ne().body, "htmx:historyCacheMiss", i);
    e.open("GET", o, true);
    e.setRequestHeader("HX-Request", "true");
    e.setRequestHeader("HX-History-Restore-Request", "true");
    e.setRequestHeader("HX-Current-URL", ne().location.href);
    e.onload = function () {
      if (this.status >= 200 && this.status < 400) {
        de(ne().body, "htmx:historyCacheMissLoad", i);
        const e = D(this.response);
        const t =
          e.querySelector("[hx-history-elt],[data-hx-history-elt]") || e;
        const n = jt();
        const r = xn(n);
        Dn(e.title);
        Ve(n, t, r);
        Gt(r.tasks);
        Ut = o;
        de(ne().body, "htmx:historyRestore", {
          path: o,
          cacheMiss: true,
          serverResponse: this.response,
        });
      } else {
        fe(ne().body, "htmx:historyCacheMissLoadError", i);
      }
    };
    e.send();
  }
  function Yt(e) {
    zt();
    e = e || location.pathname + location.search;
    const t = _t(e);
    if (t) {
      const n = D(t.content);
      const r = jt();
      const o = xn(r);
      Dn(n.title);
      Ve(r, n, o);
      Gt(o.tasks);
      E().setTimeout(function () {
        window.scrollTo(0, t.scroll);
      }, 0);
      Ut = e;
      de(ne().body, "htmx:historyRestore", { path: e, item: t });
    } else {
      if (Q.config.refreshOnHistoryMiss) {
        window.location.reload(true);
      } else {
        Zt(e);
      }
    }
  }
  function Wt(e) {
    let t = Se(e, "hx-indicator");
    if (t == null) {
      t = [e];
    }
    se(t, function (e) {
      const t = ie(e);
      t.requestCount = (t.requestCount || 0) + 1;
      e.classList.add.call(e.classList, Q.config.requestClass);
    });
    return t;
  }
  function Qt(e) {
    let t = Se(e, "hx-disabled-elt");
    if (t == null) {
      t = [];
    }
    se(t, function (e) {
      const t = ie(e);
      t.requestCount = (t.requestCount || 0) + 1;
      e.setAttribute("disabled", "");
      e.setAttribute("data-disabled-by-htmx", "");
    });
    return t;
  }
  function en(e, t) {
    se(e, function (e) {
      const t = ie(e);
      t.requestCount = (t.requestCount || 0) - 1;
      if (t.requestCount === 0) {
        e.classList.remove.call(e.classList, Q.config.requestClass);
      }
    });
    se(t, function (e) {
      const t = ie(e);
      t.requestCount = (t.requestCount || 0) - 1;
      if (t.requestCount === 0) {
        e.removeAttribute("disabled");
        e.removeAttribute("data-disabled-by-htmx");
      }
    });
  }
  function tn(t, n) {
    for (let e = 0; e < t.length; e++) {
      const r = t[e];
      if (r.isSameNode(n)) {
        return true;
      }
    }
    return false;
  }
  function nn(e) {
    const t = e;
    if (
      t.name === "" ||
      t.name == null ||
      t.disabled ||
      g(t, "fieldset[disabled]")
    ) {
      return false;
    }
    if (
      t.type === "button" ||
      t.type === "submit" ||
      t.tagName === "image" ||
      t.tagName === "reset" ||
      t.tagName === "file"
    ) {
      return false;
    }
    if (t.type === "checkbox" || t.type === "radio") {
      return t.checked;
    }
    return true;
  }
  function rn(t, e, n) {
    if (t != null && e != null) {
      if (Array.isArray(e)) {
        e.forEach(function (e) {
          n.append(t, e);
        });
      } else {
        n.append(t, e);
      }
    }
  }
  function on(t, n, r) {
    if (t != null && n != null) {
      let e = r.getAll(t);
      if (Array.isArray(n)) {
        e = e.filter((e) => n.indexOf(e) < 0);
      } else {
        e = e.filter((e) => e !== n);
      }
      r.delete(t);
      se(e, (e) => r.append(t, e));
    }
  }
  function sn(t, n, r, o, i) {
    if (o == null || tn(t, o)) {
      return;
    } else {
      t.push(o);
    }
    if (nn(o)) {
      const s = ee(o, "name");
      let e = o.value;
      if (o instanceof HTMLSelectElement && o.multiple) {
        e = F(o.querySelectorAll("option:checked")).map(function (e) {
          return e.value;
        });
      }
      if (o instanceof HTMLInputElement && o.files) {
        e = F(o.files);
      }
      rn(s, e, n);
      if (i) {
        ln(o, r);
      }
    }
    if (o instanceof HTMLFormElement) {
      se(o.elements, function (e) {
        if (t.indexOf(e) >= 0) {
          on(e.name, e.value, n);
        } else {
          t.push(e);
        }
        if (i) {
          ln(e, r);
        }
      });
      new FormData(o).forEach(function (e, t) {
        if (e instanceof File && e.name === "") {
          return;
        }
        rn(t, e, n);
      });
    }
  }
  function ln(e, t) {
    const n = e;
    if (n.willValidate) {
      de(n, "htmx:validation:validate");
      if (!n.checkValidity()) {
        t.push({ elt: n, message: n.validationMessage, validity: n.validity });
        de(n, "htmx:validation:failed", {
          message: n.validationMessage,
          validity: n.validity,
        });
      }
    }
  }
  function un(n, e) {
    for (const t of e.keys()) {
      n.delete(t);
    }
    e.forEach(function (e, t) {
      n.append(t, e);
    });
    return n;
  }
  function cn(e, t) {
    const n = [];
    const r = new FormData();
    const o = new FormData();
    const i = [];
    const s = ie(e);
    if (s.lastButtonClicked && !le(s.lastButtonClicked)) {
      s.lastButtonClicked = null;
    }
    let l =
      (e instanceof HTMLFormElement && e.noValidate !== true) ||
      te(e, "hx-validate") === "true";
    if (s.lastButtonClicked) {
      l = l && s.lastButtonClicked.formNoValidate !== true;
    }
    if (t !== "get") {
      sn(n, o, i, g(e, "form"), l);
    }
    sn(n, r, i, e, l);
    if (
      s.lastButtonClicked ||
      e.tagName === "BUTTON" ||
      (e.tagName === "INPUT" && ee(e, "type") === "submit")
    ) {
      const c = s.lastButtonClicked || e;
      const a = ee(c, "name");
      rn(a, c.value, o);
    }
    const u = Se(e, "hx-include");
    se(u, function (e) {
      sn(n, r, i, ce(e), l);
      if (!f(e, "form")) {
        se(d(e).querySelectorAll(it), function (e) {
          sn(n, r, i, e, l);
        });
      }
    });
    un(r, o);
    return { errors: i, formData: r, values: An(r) };
  }
  function an(e, t, n) {
    if (e !== "") {
      e += "&";
    }
    if (String(n) === "[object Object]") {
      n = JSON.stringify(n);
    }
    const r = encodeURIComponent(n);
    e += encodeURIComponent(t) + "=" + r;
    return e;
  }
  function fn(e) {
    e = Ln(e);
    let n = "";
    e.forEach(function (e, t) {
      n = an(n, t, e);
    });
    return n;
  }
  function dn(e, t, n) {
    const r = {
      "HX-Request": "true",
      "HX-Trigger": ee(e, "id"),
      "HX-Trigger-Name": ee(e, "name"),
      "HX-Target": te(t, "id"),
      "HX-Current-URL": ne().location.href,
    };
    wn(e, "hx-headers", false, r);
    if (n !== undefined) {
      r["HX-Prompt"] = n;
    }
    if (ie(e).boosted) {
      r["HX-Boosted"] = "true";
    }
    return r;
  }
  function hn(n, e) {
    const t = re(e, "hx-params");
    if (t) {
      if (t === "none") {
        return new FormData();
      } else if (t === "*") {
        return n;
      } else if (t.indexOf("not ") === 0) {
        se(t.substr(4).split(","), function (e) {
          e = e.trim();
          n.delete(e);
        });
        return n;
      } else {
        const r = new FormData();
        se(t.split(","), function (t) {
          t = t.trim();
          if (n.has(t)) {
            n.getAll(t).forEach(function (e) {
              r.append(t, e);
            });
          }
        });
        return r;
      }
    } else {
      return n;
    }
  }
  function gn(e) {
    return !!ee(e, "href") && ee(e, "href").indexOf("#") >= 0;
  }
  function pn(e, t) {
    const n = t || re(e, "hx-swap");
    const r = {
      swapStyle: ie(e).boosted ? "innerHTML" : Q.config.defaultSwapStyle,
      swapDelay: Q.config.defaultSwapDelay,
      settleDelay: Q.config.defaultSettleDelay,
    };
    if (Q.config.scrollIntoViewOnBoost && ie(e).boosted && !gn(e)) {
      r.show = "top";
    }
    if (n) {
      const s = U(n);
      if (s.length > 0) {
        for (let e = 0; e < s.length; e++) {
          const l = s[e];
          if (l.indexOf("swap:") === 0) {
            r.swapDelay = h(l.substr(5));
          } else if (l.indexOf("settle:") === 0) {
            r.settleDelay = h(l.substr(7));
          } else if (l.indexOf("transition:") === 0) {
            r.transition = l.substr(11) === "true";
          } else if (l.indexOf("ignoreTitle:") === 0) {
            r.ignoreTitle = l.substr(12) === "true";
          } else if (l.indexOf("scroll:") === 0) {
            const u = l.substr(7);
            var o = u.split(":");
            const c = o.pop();
            var i = o.length > 0 ? o.join(":") : null;
            r.scroll = c;
            r.scrollTarget = i;
          } else if (l.indexOf("show:") === 0) {
            const a = l.substr(5);
            var o = a.split(":");
            const f = o.pop();
            var i = o.length > 0 ? o.join(":") : null;
            r.show = f;
            r.showTarget = i;
          } else if (l.indexOf("focus-scroll:") === 0) {
            const d = l.substr("focus-scroll:".length);
            r.focusScroll = d == "true";
          } else if (e == 0) {
            r.swapStyle = l;
          } else {
            w("Unknown modifier in hx-swap: " + l);
          }
        }
      }
    }
    return r;
  }
  function mn(e) {
    return (
      re(e, "hx-encoding") === "multipart/form-data" ||
      (f(e, "form") && ee(e, "enctype") === "multipart/form-data")
    );
  }
  function yn(t, n, r) {
    let o = null;
    Bt(n, function (e) {
      if (o == null) {
        o = e.encodeParameters(t, r, n);
      }
    });
    if (o != null) {
      return o;
    } else {
      if (mn(n)) {
        return un(new FormData(), Ln(r));
      } else {
        return fn(r);
      }
    }
  }
  function xn(e) {
    return { tasks: [], elts: [e] };
  }
  function bn(e, t) {
    const n = e[0];
    const r = e[e.length - 1];
    if (t.scroll) {
      var o = null;
      if (t.scrollTarget) {
        o = ce(ae(n, t.scrollTarget));
      }
      if (t.scroll === "top" && (n || o)) {
        o = o || n;
        o.scrollTop = 0;
      }
      if (t.scroll === "bottom" && (r || o)) {
        o = o || r;
        o.scrollTop = o.scrollHeight;
      }
    }
    if (t.show) {
      var o = null;
      if (t.showTarget) {
        let e = t.showTarget;
        if (t.showTarget === "window") {
          e = "body";
        }
        o = ce(ae(n, e));
      }
      if (t.show === "top" && (n || o)) {
        o = o || n;
        o.scrollIntoView({ block: "start", behavior: Q.config.scrollBehavior });
      }
      if (t.show === "bottom" && (r || o)) {
        o = o || r;
        o.scrollIntoView({ block: "end", behavior: Q.config.scrollBehavior });
      }
    }
  }
  function wn(r, e, o, i) {
    if (i == null) {
      i = {};
    }
    if (r == null) {
      return i;
    }
    const s = te(r, e);
    if (s) {
      let e = s.trim();
      let t = o;
      if (e === "unset") {
        return null;
      }
      if (e.indexOf("javascript:") === 0) {
        e = e.substr(11);
        t = true;
      } else if (e.indexOf("js:") === 0) {
        e = e.substr(3);
        t = true;
      }
      if (e.indexOf("{") !== 0) {
        e = "{" + e + "}";
      }
      let n;
      if (t) {
        n = vn(
          r,
          function () {
            return Function("return (" + e + ")")();
          },
          {},
        );
      } else {
        n = S(e);
      }
      for (const l in n) {
        if (n.hasOwnProperty(l)) {
          if (i[l] == null) {
            i[l] = n[l];
          }
        }
      }
    }
    return wn(ce(u(r)), e, o, i);
  }
  function vn(e, t, n) {
    if (Q.config.allowEval) {
      return t();
    } else {
      fe(e, "htmx:evalDisallowedError");
      return n;
    }
  }
  function Sn(e, t) {
    return wn(e, "hx-vars", true, t);
  }
  function En(e, t) {
    return wn(e, "hx-vals", false, t);
  }
  function Cn(e) {
    return ue(Sn(e), En(e));
  }
  function On(t, n, r) {
    if (r !== null) {
      try {
        t.setRequestHeader(n, r);
      } catch (e) {
        t.setRequestHeader(n, encodeURIComponent(r));
        t.setRequestHeader(n + "-URI-AutoEncoded", "true");
      }
    }
  }
  function Rn(t) {
    if (t.responseURL && typeof URL !== "undefined") {
      try {
        const e = new URL(t.responseURL);
        return e.pathname + e.search;
      } catch (e) {
        fe(ne().body, "htmx:badResponseUrl", { url: t.responseURL });
      }
    }
  }
  function C(e, t) {
    return t.test(e.getAllResponseHeaders());
  }
  function Hn(e, t, n) {
    e = e.toLowerCase();
    if (n) {
      if (n instanceof Element || typeof n === "string") {
        return he(e, t, null, null, {
          targetOverride: y(n),
          returnPromise: true,
        });
      } else {
        return he(e, t, y(n.source), n.event, {
          handler: n.handler,
          headers: n.headers,
          values: n.values,
          targetOverride: y(n.target),
          swapOverride: n.swap,
          select: n.select,
          returnPromise: true,
        });
      }
    } else {
      return he(e, t, null, null, { returnPromise: true });
    }
  }
  function Tn(e) {
    const t = [];
    while (e) {
      t.push(e);
      e = e.parentElement;
    }
    return t;
  }
  function qn(e, t, n) {
    let r;
    let o;
    if (typeof URL === "function") {
      o = new URL(t, document.location.href);
      const i = document.location.origin;
      r = i === o.origin;
    } else {
      o = t;
      r = l(t, document.location.origin);
    }
    if (Q.config.selfRequestsOnly) {
      if (!r) {
        return false;
      }
    }
    return de(e, "htmx:validateUrl", ue({ url: o, sameHost: r }, n));
  }
  function Ln(e) {
    if (e instanceof FormData) return e;
    const t = new FormData();
    for (const n in e) {
      if (e.hasOwnProperty(n)) {
        if (typeof e[n].forEach === "function") {
          e[n].forEach(function (e) {
            t.append(n, e);
          });
        } else if (typeof e[n] === "object" && !(e[n] instanceof Blob)) {
          t.append(n, JSON.stringify(e[n]));
        } else {
          t.append(n, e[n]);
        }
      }
    }
    return t;
  }
  function Nn(r, o, e) {
    return new Proxy(e, {
      get: function (t, e) {
        if (typeof e === "number") return t[e];
        if (e === "length") return t.length;
        if (e === "push") {
          return function (e) {
            t.push(e);
            r.append(o, e);
          };
        }
        if (typeof t[e] === "function") {
          return function () {
            t[e].apply(t, arguments);
            r.delete(o);
            t.forEach(function (e) {
              r.append(o, e);
            });
          };
        }
        if (t[e] && t[e].length === 1) {
          return t[e][0];
        } else {
          return t[e];
        }
      },
      set: function (e, t, n) {
        e[t] = n;
        r.delete(o);
        e.forEach(function (e) {
          r.append(o, e);
        });
        return true;
      },
    });
  }
  function An(r) {
    return new Proxy(r, {
      get: function (e, t) {
        if (typeof t === "symbol") {
          return Reflect.get(e, t);
        }
        if (t === "toJSON") {
          return () => Object.fromEntries(r);
        }
        if (t in e) {
          if (typeof e[t] === "function") {
            return function () {
              return r[t].apply(r, arguments);
            };
          } else {
            return e[t];
          }
        }
        const n = r.getAll(t);
        if (n.length === 0) {
          return undefined;
        } else if (n.length === 1) {
          return n[0];
        } else {
          return Nn(e, t, n);
        }
      },
      set: function (t, n, e) {
        if (typeof n !== "string") {
          return false;
        }
        t.delete(n);
        if (typeof e.forEach === "function") {
          e.forEach(function (e) {
            t.append(n, e);
          });
        } else if (typeof e === "object" && !(e instanceof Blob)) {
          t.append(n, JSON.stringify(e));
        } else {
          t.append(n, e);
        }
        return true;
      },
      deleteProperty: function (e, t) {
        if (typeof t === "string") {
          e.delete(t);
        }
        return true;
      },
      ownKeys: function (e) {
        return Reflect.ownKeys(Object.fromEntries(e));
      },
      getOwnPropertyDescriptor: function (e, t) {
        return Reflect.getOwnPropertyDescriptor(Object.fromEntries(e), t);
      },
    });
  }
  function he(t, n, r, o, i, D) {
    let s = null;
    let l = null;
    i = i != null ? i : {};
    if (i.returnPromise && typeof Promise !== "undefined") {
      var e = new Promise(function (e, t) {
        s = e;
        l = t;
      });
    }
    if (r == null) {
      r = ne().body;
    }
    const M = i.handler || Mn;
    const X = i.select || null;
    if (!le(r)) {
      oe(s);
      return e;
    }
    const u = i.targetOverride || ce(Ce(r));
    if (u == null || u == ve) {
      fe(r, "htmx:targetError", { target: te(r, "hx-target") });
      oe(l);
      return e;
    }
    let c = ie(r);
    const a = c.lastButtonClicked;
    if (a) {
      const L = ee(a, "formaction");
      if (L != null) {
        n = L;
      }
      const N = ee(a, "formmethod");
      if (N != null) {
        if (N.toLowerCase() !== "dialog") {
          t = N;
        }
      }
    }
    const f = re(r, "hx-confirm");
    if (D === undefined) {
      const K = function (e) {
        return he(t, n, r, o, i, !!e);
      };
      const G = {
        target: u,
        elt: r,
        path: n,
        verb: t,
        triggeringEvent: o,
        etc: i,
        issueRequest: K,
        question: f,
      };
      if (de(r, "htmx:confirm", G) === false) {
        oe(s);
        return e;
      }
    }
    let d = r;
    let h = re(r, "hx-sync");
    let g = null;
    let F = false;
    if (h) {
      const A = h.split(":");
      const I = A[0].trim();
      if (I === "this") {
        d = Ee(r, "hx-sync");
      } else {
        d = ce(ae(r, I));
      }
      h = (A[1] || "drop").trim();
      c = ie(d);
      if (h === "drop" && c.xhr && c.abortable !== true) {
        oe(s);
        return e;
      } else if (h === "abort") {
        if (c.xhr) {
          oe(s);
          return e;
        } else {
          F = true;
        }
      } else if (h === "replace") {
        de(d, "htmx:abort");
      } else if (h.indexOf("queue") === 0) {
        const Z = h.split(" ");
        g = (Z[1] || "last").trim();
      }
    }
    if (c.xhr) {
      if (c.abortable) {
        de(d, "htmx:abort");
      } else {
        if (g == null) {
          if (o) {
            const P = ie(o);
            if (P && P.triggerSpec && P.triggerSpec.queue) {
              g = P.triggerSpec.queue;
            }
          }
          if (g == null) {
            g = "last";
          }
        }
        if (c.queuedRequests == null) {
          c.queuedRequests = [];
        }
        if (g === "first" && c.queuedRequests.length === 0) {
          c.queuedRequests.push(function () {
            he(t, n, r, o, i);
          });
        } else if (g === "all") {
          c.queuedRequests.push(function () {
            he(t, n, r, o, i);
          });
        } else if (g === "last") {
          c.queuedRequests = [];
          c.queuedRequests.push(function () {
            he(t, n, r, o, i);
          });
        }
        oe(s);
        return e;
      }
    }
    const p = new XMLHttpRequest();
    c.xhr = p;
    c.abortable = F;
    const m = function () {
      c.xhr = null;
      c.abortable = false;
      if (c.queuedRequests != null && c.queuedRequests.length > 0) {
        const e = c.queuedRequests.shift();
        e();
      }
    };
    const B = re(r, "hx-prompt");
    if (B) {
      var y = prompt(B);
      if (y === null || !de(r, "htmx:prompt", { prompt: y, target: u })) {
        oe(s);
        m();
        return e;
      }
    }
    if (f && !D) {
      if (!confirm(f)) {
        oe(s);
        m();
        return e;
      }
    }
    let x = dn(r, u, y);
    if (t !== "get" && !mn(r)) {
      x["Content-Type"] = "application/x-www-form-urlencoded";
    }
    if (i.headers) {
      x = ue(x, i.headers);
    }
    const U = cn(r, t);
    let b = U.errors;
    const j = U.formData;
    if (i.values) {
      un(j, Ln(i.values));
    }
    const V = Ln(Cn(r));
    const w = un(j, V);
    let v = hn(w, r);
    if (Q.config.getCacheBusterParam && t === "get") {
      v.set("org.htmx.cache-buster", ee(u, "id") || "true");
    }
    if (n == null || n === "") {
      n = ne().location.href;
    }
    const S = wn(r, "hx-request");
    const _ = ie(r).boosted;
    let E = Q.config.methodsThatUseUrlParams.indexOf(t) >= 0;
    const C = {
      boosted: _,
      useUrlParams: E,
      formData: v,
      parameters: An(v),
      unfilteredFormData: w,
      unfilteredParameters: An(w),
      headers: x,
      target: u,
      verb: t,
      errors: b,
      withCredentials:
        i.credentials || S.credentials || Q.config.withCredentials,
      timeout: i.timeout || S.timeout || Q.config.timeout,
      path: n,
      triggeringEvent: o,
    };
    if (!de(r, "htmx:configRequest", C)) {
      oe(s);
      m();
      return e;
    }
    n = C.path;
    t = C.verb;
    x = C.headers;
    v = Ln(C.parameters);
    b = C.errors;
    E = C.useUrlParams;
    if (b && b.length > 0) {
      de(r, "htmx:validation:halted", C);
      oe(s);
      m();
      return e;
    }
    const $ = n.split("#");
    const z = $[0];
    const O = $[1];
    let R = n;
    if (E) {
      R = z;
      const Y = !v.keys().next().done;
      if (Y) {
        if (R.indexOf("?") < 0) {
          R += "?";
        } else {
          R += "&";
        }
        R += fn(v);
        if (O) {
          R += "#" + O;
        }
      }
    }
    if (!qn(r, R, C)) {
      fe(r, "htmx:invalidPath", C);
      oe(l);
      return e;
    }
    p.open(t.toUpperCase(), R, true);
    p.overrideMimeType("text/html");
    p.withCredentials = C.withCredentials;
    p.timeout = C.timeout;
    if (S.noHeaders) {
    } else {
      for (const k in x) {
        if (x.hasOwnProperty(k)) {
          const W = x[k];
          On(p, k, W);
        }
      }
    }
    const H = {
      xhr: p,
      target: u,
      requestConfig: C,
      etc: i,
      boosted: _,
      select: X,
      pathInfo: {
        requestPath: n,
        finalRequestPath: R,
        responsePath: null,
        anchor: O,
      },
    };
    p.onload = function () {
      try {
        const t = Tn(r);
        H.pathInfo.responsePath = Rn(p);
        M(r, H);
        if (H.keepIndicators !== true) {
          en(T, q);
        }
        de(r, "htmx:afterRequest", H);
        de(r, "htmx:afterOnLoad", H);
        if (!le(r)) {
          let e = null;
          while (t.length > 0 && e == null) {
            const n = t.shift();
            if (le(n)) {
              e = n;
            }
          }
          if (e) {
            de(e, "htmx:afterRequest", H);
            de(e, "htmx:afterOnLoad", H);
          }
        }
        oe(s);
        m();
      } catch (e) {
        fe(r, "htmx:onLoadError", ue({ error: e }, H));
        throw e;
      }
    };
    p.onerror = function () {
      en(T, q);
      fe(r, "htmx:afterRequest", H);
      fe(r, "htmx:sendError", H);
      oe(l);
      m();
    };
    p.onabort = function () {
      en(T, q);
      fe(r, "htmx:afterRequest", H);
      fe(r, "htmx:sendAbort", H);
      oe(l);
      m();
    };
    p.ontimeout = function () {
      en(T, q);
      fe(r, "htmx:afterRequest", H);
      fe(r, "htmx:timeout", H);
      oe(l);
      m();
    };
    if (!de(r, "htmx:beforeRequest", H)) {
      oe(s);
      m();
      return e;
    }
    var T = Wt(r);
    var q = Qt(r);
    se(["loadstart", "loadend", "progress", "abort"], function (t) {
      se([p, p.upload], function (e) {
        e.addEventListener(t, function (e) {
          de(r, "htmx:xhr:" + t, {
            lengthComputable: e.lengthComputable,
            loaded: e.loaded,
            total: e.total,
          });
        });
      });
    });
    de(r, "htmx:beforeSend", H);
    const J = E ? null : yn(p, r, v);
    p.send(J);
    return e;
  }
  function In(e, t) {
    const n = t.xhr;
    let r = null;
    let o = null;
    if (C(n, /HX-Push:/i)) {
      r = n.getResponseHeader("HX-Push");
      o = "push";
    } else if (C(n, /HX-Push-Url:/i)) {
      r = n.getResponseHeader("HX-Push-Url");
      o = "push";
    } else if (C(n, /HX-Replace-Url:/i)) {
      r = n.getResponseHeader("HX-Replace-Url");
      o = "replace";
    }
    if (r) {
      if (r === "false") {
        return {};
      } else {
        return { type: o, path: r };
      }
    }
    const i = t.pathInfo.finalRequestPath;
    const s = t.pathInfo.responsePath;
    const l = re(e, "hx-push-url");
    const u = re(e, "hx-replace-url");
    const c = ie(e).boosted;
    let a = null;
    let f = null;
    if (l) {
      a = "push";
      f = l;
    } else if (u) {
      a = "replace";
      f = u;
    } else if (c) {
      a = "push";
      f = s || i;
    }
    if (f) {
      if (f === "false") {
        return {};
      }
      if (f === "true") {
        f = s || i;
      }
      if (t.pathInfo.anchor && f.indexOf("#") === -1) {
        f = f + "#" + t.pathInfo.anchor;
      }
      return { type: a, path: f };
    } else {
      return {};
    }
  }
  function Pn(e, t) {
    var n = new RegExp(e.code);
    return n.test(t.toString(10));
  }
  function kn(e) {
    for (var t = 0; t < Q.config.responseHandling.length; t++) {
      var n = Q.config.responseHandling[t];
      if (Pn(n, e.status)) {
        return n;
      }
    }
    return { swap: false };
  }
  function Dn(e) {
    if (e) {
      const t = r("title");
      if (t) {
        t.innerHTML = e;
      } else {
        window.document.title = e;
      }
    }
  }
  function Mn(o, i) {
    const s = i.xhr;
    let l = i.target;
    const e = i.etc;
    const u = i.select;
    if (!de(o, "htmx:beforeOnLoad", i)) return;
    if (C(s, /HX-Trigger:/i)) {
      Je(s, "HX-Trigger", o);
    }
    if (C(s, /HX-Location:/i)) {
      zt();
      let e = s.getResponseHeader("HX-Location");
      var t;
      if (e.indexOf("{") === 0) {
        t = S(e);
        e = t.path;
        delete t.path;
      }
      Hn("get", e, t).then(function () {
        Jt(e);
      });
      return;
    }
    const n =
      C(s, /HX-Refresh:/i) && s.getResponseHeader("HX-Refresh") === "true";
    if (C(s, /HX-Redirect:/i)) {
      i.keepIndicators = true;
      location.href = s.getResponseHeader("HX-Redirect");
      n && location.reload();
      return;
    }
    if (n) {
      i.keepIndicators = true;
      location.reload();
      return;
    }
    if (C(s, /HX-Retarget:/i)) {
      if (s.getResponseHeader("HX-Retarget") === "this") {
        i.target = o;
      } else {
        i.target = ce(ae(o, s.getResponseHeader("HX-Retarget")));
      }
    }
    const c = In(o, i);
    const r = kn(s);
    const a = r.swap;
    let f = !!r.error;
    let d = Q.config.ignoreTitle || r.ignoreTitle;
    let h = r.select;
    if (r.target) {
      i.target = ce(ae(o, r.target));
    }
    var g = e.swapOverride;
    if (g == null && r.swapOverride) {
      g = r.swapOverride;
    }
    if (C(s, /HX-Retarget:/i)) {
      if (s.getResponseHeader("HX-Retarget") === "this") {
        i.target = o;
      } else {
        i.target = ce(ae(o, s.getResponseHeader("HX-Retarget")));
      }
    }
    if (C(s, /HX-Reswap:/i)) {
      g = s.getResponseHeader("HX-Reswap");
    }
    var p = s.response;
    var m = ue(
      {
        shouldSwap: a,
        serverResponse: p,
        isError: f,
        ignoreTitle: d,
        selectOverride: h,
      },
      i,
    );
    if (r.event && !de(l, r.event, m)) return;
    if (!de(l, "htmx:beforeSwap", m)) return;
    l = m.target;
    p = m.serverResponse;
    f = m.isError;
    d = m.ignoreTitle;
    h = m.selectOverride;
    i.target = l;
    i.failed = f;
    i.successful = !f;
    if (m.shouldSwap) {
      if (s.status === 286) {
        ut(o);
      }
      Bt(o, function (e) {
        p = e.transformResponse(p, s, o);
      });
      if (c.type) {
        zt();
      }
      if (C(s, /HX-Reswap:/i)) {
        g = s.getResponseHeader("HX-Reswap");
      }
      var y = pn(o, g);
      if (!y.hasOwnProperty("ignoreTitle")) {
        y.ignoreTitle = d;
      }
      l.classList.add(Q.config.swappingClass);
      let n = null;
      let r = null;
      if (u) {
        h = u;
      }
      if (C(s, /HX-Reselect:/i)) {
        h = s.getResponseHeader("HX-Reselect");
      }
      const x = re(o, "hx-select-oob");
      const b = re(o, "hx-select");
      let e = function () {
        try {
          if (c.type) {
            de(ne().body, "htmx:beforeHistoryUpdate", ue({ history: c }, i));
            if (c.type === "push") {
              Jt(c.path);
              de(ne().body, "htmx:pushedIntoHistory", { path: c.path });
            } else {
              Kt(c.path);
              de(ne().body, "htmx:replacedInHistory", { path: c.path });
            }
          }
          ze(l, p, y, {
            select: h || b,
            selectOOB: x,
            eventInfo: i,
            anchor: i.pathInfo.anchor,
            contextElement: o,
            afterSwapCallback: function () {
              if (C(s, /HX-Trigger-After-Swap:/i)) {
                let e = o;
                if (!le(o)) {
                  e = ne().body;
                }
                Je(s, "HX-Trigger-After-Swap", e);
              }
            },
            afterSettleCallback: function () {
              if (C(s, /HX-Trigger-After-Settle:/i)) {
                let e = o;
                if (!le(o)) {
                  e = ne().body;
                }
                Je(s, "HX-Trigger-After-Settle", e);
              }
              oe(n);
            },
          });
        } catch (e) {
          fe(o, "htmx:swapError", i);
          oe(r);
          throw e;
        }
      };
      let t = Q.config.globalViewTransitions;
      if (y.hasOwnProperty("transition")) {
        t = y.transition;
      }
      if (
        t &&
        de(o, "htmx:beforeTransition", i) &&
        typeof Promise !== "undefined" &&
        document.startViewTransition
      ) {
        const w = new Promise(function (e, t) {
          n = e;
          r = t;
        });
        const v = e;
        e = function () {
          document.startViewTransition(function () {
            v();
            return w;
          });
        };
      }
      if (y.swapDelay > 0) {
        E().setTimeout(e, y.swapDelay);
      } else {
        e();
      }
    }
    if (f) {
      fe(
        o,
        "htmx:responseError",
        ue(
          {
            error:
              "Response Status Error Code " +
              s.status +
              " from " +
              i.pathInfo.requestPath,
          },
          i,
        ),
      );
    }
  }
  const Xn = {};
  function Fn() {
    return {
      init: function (e) {
        return null;
      },
      getSelectors: function () {
        return null;
      },
      onEvent: function (e, t) {
        return true;
      },
      transformResponse: function (e, t, n) {
        return e;
      },
      isInlineSwap: function (e) {
        return false;
      },
      handleSwap: function (e, t, n, r) {
        return false;
      },
      encodeParameters: function (e, t, n) {
        return null;
      },
    };
  }
  function Bn(e, t) {
    if (t.init) {
      t.init(n);
    }
    Xn[e] = ue(Fn(), t);
  }
  function Un(e) {
    delete Xn[e];
  }
  function jn(e, n, r) {
    if (n == undefined) {
      n = [];
    }
    if (e == undefined) {
      return n;
    }
    if (r == undefined) {
      r = [];
    }
    const t = te(e, "hx-ext");
    if (t) {
      se(t.split(","), function (e) {
        e = e.replace(/ /g, "");
        if (e.slice(0, 7) == "ignore:") {
          r.push(e.slice(7));
          return;
        }
        if (r.indexOf(e) < 0) {
          const t = Xn[e];
          if (t && n.indexOf(t) < 0) {
            n.push(t);
          }
        }
      });
    }
    return jn(ce(u(e)), n, r);
  }
  var Vn = false;
  ne().addEventListener("DOMContentLoaded", function () {
    Vn = true;
  });
  function _n(e) {
    if (Vn || ne().readyState === "complete") {
      e();
    } else {
      ne().addEventListener("DOMContentLoaded", e);
    }
  }
  function $n() {
    if (Q.config.includeIndicatorStyles !== false) {
      const e = Q.config.inlineStyleNonce
        ? ` nonce="${Q.config.inlineStyleNonce}"`
        : "";
      ne().head.insertAdjacentHTML(
        "beforeend",
        "<style" +
          e +
          ">      ." +
          Q.config.indicatorClass +
          "{opacity:0}      ." +
          Q.config.requestClass +
          " ." +
          Q.config.indicatorClass +
          "{opacity:1; transition: opacity 200ms ease-in;}      ." +
          Q.config.requestClass +
          "." +
          Q.config.indicatorClass +
          "{opacity:1; transition: opacity 200ms ease-in;}      </style>",
      );
    }
  }
  function zn() {
    const e = ne().querySelector('meta[name="htmx-config"]');
    if (e) {
      return S(e.content);
    } else {
      return null;
    }
  }
  function Jn() {
    const e = zn();
    if (e) {
      Q.config = ue(Q.config, e);
    }
  }
  _n(function () {
    Jn();
    $n();
    let e = ne().body;
    Dt(e);
    const t = ne().querySelectorAll(
      "[hx-trigger='restored'],[data-hx-trigger='restored']",
    );
    e.addEventListener("htmx:abort", function (e) {
      const t = e.target;
      const n = ie(t);
      if (n && n.xhr) {
        n.xhr.abort();
      }
    });
    const n = window.onpopstate ? window.onpopstate.bind(window) : null;
    window.onpopstate = function (e) {
      if (e.state && e.state.htmx) {
        Yt();
        se(t, function (e) {
          de(e, "htmx:restored", { document: ne(), triggerEvent: de });
        });
      } else {
        if (n) {
          n(e);
        }
      }
    };
    E().setTimeout(function () {
      de(e, "htmx:load", {});
      e = null;
    }, 0);
  });
  return Q;
})();
