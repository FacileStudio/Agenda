(function () {
  if (location.protocol !== "http:" && location.protocol !== "https:") return;
  var host = location.hostname;
  if (host === "localhost" || host === "127.0.0.1" || /\.localhost$/.test(host)) return;

  var script = document.currentScript;
  var mask = script.hasAttribute("data-mask");
  var excludes = (script.getAttribute("data-exclude") || "").split(",").map(function (prefix) {
    return prefix.trim();
  }).filter(Boolean);
  var base = "https://vision.facile.studio/api/e";
  var endpoint = base + "/p";
  var heartbeatEndpoint = base + "/h";
  var flags = (navigator.webdriver === true ? 1 : 0) | (window.outerWidth === 0 && window.outerHeight === 0 ? 2 : 0);

  function visitorId() {
    var key = "_vs_id";
    var id = localStorage.getItem(key);
    if (id) return id;
    id = Math.random().toString(36).substring(2) + Date.now().toString(36);
    try { localStorage.setItem(key, id); } catch (e) {}
    return id;
  }

  var uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

  function isIdentifier(segment) {
    var plain = segment;
    try { plain = decodeURIComponent(segment); } catch (e) {}
    if (/^\d+$/.test(plain) || uuid.test(plain) || plain.indexOf("@") !== -1) return true;
    return plain.length >= 16 && /\d/.test(plain);
  }

  function cleanPath(path) {
    if (!mask) return path;
    return path.split("/").map(function (segment) {
      return segment && isIdentifier(segment) ? ":id" : segment;
    }).join("/");
  }

  function cleanReferrer(referrer) {
    if (!mask || !referrer) return referrer;
    try {
      var url = new URL(referrer);
      if (url.origin !== location.origin) return url.origin;
      return url.origin + cleanPath(url.pathname);
    } catch (e) { return ""; }
  }

  function excluded() {
    var path = location.pathname;
    for (var i = 0; i < excludes.length; i++) {
      if (path.indexOf(excludes[i]) === 0) return true;
    }
    return false;
  }

  function getUTM() {
    var params = new URLSearchParams(location.search);
    return {
      utm_source: params.get('utm_source') || '',
      utm_medium: params.get('utm_medium') || '',
      utm_campaign: params.get('utm_campaign') || '',
      utm_term: params.get('utm_term') || '',
      utm_content: params.get('utm_content') || ''
    };
  }

  function send(path) {
    if (excluded()) return;
    var utm = getUTM();
    var data = {
      hostname: location.hostname,
      path: cleanPath(path || location.pathname),
      referrer: cleanReferrer(document.referrer || ""),
      language: navigator.language || "",
      screen_width: window.innerWidth,
      visitor_id: visitorId(),
      b: flags,
      utm_source: utm.utm_source,
      utm_medium: utm.utm_medium,
      utm_campaign: utm.utm_campaign,
      utm_term: utm.utm_term,
      utm_content: utm.utm_content
    };
    var img = new Image();
    img.src = endpoint + "?data=" + encodeURIComponent(JSON.stringify(data));
  }

  function heartbeat(interacted) {
    if (excluded()) return;
    var data = {
      hostname: location.hostname,
      visitor_id: visitorId(),
      b: flags
    };
    if (interacted) data.interacted = true;
    var img = new Image();
    img.src = heartbeatEndpoint + "?data=" + encodeURIComponent(JSON.stringify(data));
  }

  var hbTimer = null;

  function startHeartbeat() {
    if (hbTimer) return;
    heartbeat();
    hbTimer = setInterval(function () { heartbeat(); }, 30000);
  }

  function stopHeartbeat() {
    if (hbTimer) {
      clearInterval(hbTimer);
      hbTimer = null;
    }
  }

  function getPerformance() {
    try {
      var nav = performance.getEntriesByType('navigation')[0];
      if (!nav) return null;
      return {
        dns: Math.round(nav.domainLookupEnd - nav.domainLookupStart),
        tcp: Math.round(nav.connectEnd - nav.connectStart),
        ttfb: Math.round(nav.responseStart - nav.requestStart),
        dom_load: Math.round(nav.domContentLoadedEventEnd - nav.startTime),
        page_load: Math.round(nav.loadEventEnd - nav.startTime)
      };
    } catch (e) { return null; }
  }

  function sendPerformance() {
    if (excluded()) return;
    var perf = getPerformance();
    if (!perf || perf.page_load <= 0) return;
    var data = {
      hostname: location.hostname,
      path: cleanPath(location.pathname),
      visitor_id: visitorId(),
      performance: perf
    };
    var img = new Image();
    img.src = endpoint + "?data=" + encodeURIComponent(JSON.stringify(data)) + "&type=perf";
  }

  window.vision = {
    track: function (name, props) {
      if (excluded()) return;
      var data = {
        hostname: location.hostname,
        path: cleanPath(location.pathname),
        visitor_id: visitorId(),
        b: flags,
        event_name: name,
        event_props: props || {}
      };
      var img = new Image();
      img.src = endpoint.replace('/e/p', '/e/t') + "?data=" + encodeURIComponent(JSON.stringify(data));
    }
  };

  send();
  startHeartbeat();

  var gestures = ["pointerdown", "keydown", "touchstart", "wheel"];
  var gestureOpts = { capture: true, passive: true };

  function engaged() {
    for (var i = 0; i < gestures.length; i++) {
      window.removeEventListener(gestures[i], engaged, gestureOpts);
    }
    heartbeat(true);
  }

  for (var g = 0; g < gestures.length; g++) {
    window.addEventListener(gestures[g], engaged, gestureOpts);
  }

  if (document.readyState === 'complete') {
    setTimeout(sendPerformance, 100);
  } else {
    window.addEventListener('load', function () {
      setTimeout(sendPerformance, 100);
    });
  }

  document.addEventListener("visibilitychange", function () {
    if (document.hidden) {
      stopHeartbeat();
    } else {
      startHeartbeat();
    }
  });

  var pushState = history.pushState;
  history.pushState = function () {
    pushState.apply(history, arguments);
    send();
  };

  window.addEventListener("popstate", function () {
    send();
  });
})();
