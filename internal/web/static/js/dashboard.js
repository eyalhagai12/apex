(function () {
  var CHART_HEIGHT = 480;

  var INTERVAL_SECONDS = {
    "1Min": 60,
    "5Min": 300,
    "15Min": 900,
    "1H": 3600,
    "1D": 86400,
  };

  function bucketStart(unixSeconds, intervalSeconds) {
    return Math.floor(unixSeconds / intervalSeconds) * intervalSeconds;
  }

  // Live SSE ticks are always raw 1-minute bars. When a panel is displaying
  // a coarser interval, fold each tick into the currently forming bucket
  // (extend it) or roll over to a new one, so the chart keeps updating live
  // regardless of the selected interval.
  function applyLiveBar(panel, raw) {
    var intervalSeconds = INTERVAL_SECONDS[panel.dataset.timeframe] || 60;
    var time = bucketStart(raw.time, intervalSeconds);

    var last = panel._lastBar;
    var merged;
    if (last && last.time === time) {
      merged = {
        time: time,
        open: last.open,
        high: Math.max(last.high, raw.high),
        low: Math.min(last.low, raw.low),
        close: raw.close,
      };
    } else {
      merged = {
        time: time,
        open: raw.open,
        high: raw.high,
        low: raw.low,
        close: raw.close,
      };
    }
    panel._lastBar = merged;
    panel._series.update(merged);
  }

  function themeColor(name, fallback) {
    var v = getComputedStyle(document.documentElement)
      .getPropertyValue(name)
      .trim();
    return v || fallback;
  }

  function showSpinner(panel) {
    var container = panel.querySelector(".chart-container");
    if (!container || container.querySelector(".spinner-overlay")) {
      return;
    }
    var overlay = document.createElement("div");
    overlay.className =
      "spinner-overlay absolute inset-0 flex items-center justify-center bg-surface z-10";
    var spinner = document.createElement("div");
    spinner.className =
      "w-8 h-8 rounded-full border-[3px] border-border border-t-primary animate-spin";
    overlay.appendChild(spinner);
    container.appendChild(overlay);
  }

  function hideSpinner(panel) {
    var container = panel.querySelector(".chart-container");
    var overlay = container && container.querySelector(".spinner-overlay");
    if (overlay) {
      overlay.remove();
    }
  }

  function hydrate(panel) {
    var symbol = panel.dataset.symbol;
    var tf = panel.dataset.timeframe;
    showSpinner(panel);
    return fetch(
      "/web/bars?symbol=" +
        encodeURIComponent(symbol) +
        "&timeframe=" +
        encodeURIComponent(tf),
    )
      .then(function (res) {
        return res.ok ? res.json() : [];
      })
      .then(function (bars) {
        panel._series.setData(bars);
        panel._chart.timeScale().fitContent();
        panel._lastBar = bars.length ? bars[bars.length - 1] : null;
      })
      .finally(function () {
        hideSpinner(panel);
      });
  }

  function teardownPanel(panel, message) {
    if (panel._eventSource) {
      panel._eventSource.close();
    }
    if (panel._resizeObserver) {
      panel._resizeObserver.disconnect();
    }

    var alert = document.createElement("div");
    alert.className =
      "px-4 py-2 rounded-md text-sm border border-red-500 bg-red-50 text-red-700";
    alert.textContent =
      panel.dataset.symbol +
      " " +
      panel.dataset.timeframe +
      ": " +
      message +
      " ";

    var dismiss = document.createElement("button");
    dismiss.type = "button";
    dismiss.className =
      "inline-flex items-center justify-center gap-2 px-4 py-2 rounded-md text-sm font-medium bg-surface text-text border border-border hover:bg-bg-subtle transition-colors ml-2";
    dismiss.textContent = "Dismiss";
    dismiss.addEventListener("click", function () {
      panel.remove();
    });
    alert.appendChild(dismiss);

    var container = panel.querySelector(".chart-container");
    if (container) {
      container.replaceWith(alert);
    } else {
      panel.appendChild(alert);
    }
  }

  function connectEvents(panel) {
    showSpinner(panel);
    var symbol = panel.dataset.symbol;
    var es = new EventSource(
      "/web/events?symbol=" + encodeURIComponent(symbol),
    );
    es.addEventListener("bar", function (evt) {
      applyLiveBar(panel, JSON.parse(evt.data));
    });
    es.addEventListener("backfill_complete", function (evt) {
      var payload = {};
      try {
        payload = JSON.parse(evt.data);
      } catch (e) {
        payload = {};
      }
      if (payload.error) {
        teardownPanel(panel, payload.error);
        return;
      }
      hydrate(panel);
    });
    panel._eventSource = es;
  }

  function initChart(panel) {
    var container = panel.querySelector(".chart-container");
    if (!container) {
      return;
    }

    var chart = LightweightCharts.createChart(container, {
      width: container.clientWidth,
      height: CHART_HEIGHT,
      layout: {
        background: { color: themeColor("--color-surface", "#ffffff") },
        textColor: themeColor("--color-text", "#212529"),
      },
      grid: {
        vertLines: { color: themeColor("--color-border", "#e9ecef") },
        horzLines: { color: themeColor("--color-border", "#e9ecef") },
      },
      timeScale: { timeVisible: true, secondsVisible: false },
    });

    var series = chart.addCandlestickSeries({
      upColor: themeColor("--color-chart-up", "#16a34a"),
      downColor: themeColor("--color-chart-down", "#dc2626"),
      borderVisible: false,
      wickUpColor: themeColor("--color-chart-up", "#16a34a"),
      wickDownColor: themeColor("--color-chart-down", "#dc2626"),
    });

    panel._chart = chart;
    panel._series = series;

    var resizeObserver = new ResizeObserver(function (entries) {
      entries.forEach(function (entry) {
        chart.applyOptions({ width: entry.contentRect.width });
      });
    });
    resizeObserver.observe(container);
    panel._resizeObserver = resizeObserver;

    var intervalSelect = panel.querySelector(".interval-select");
    if (intervalSelect) {
      intervalSelect.addEventListener("change", function () {
        panel.dataset.timeframe = intervalSelect.value;
        panel._lastBar = null;
        hydrate(panel);
      });
    }

    hydrate(panel).then(function () {
      connectEvents(panel);
    });
  }

  function findNewPanels(node) {
    if (!node) {
      return [];
    }
    if (node.matches && node.matches(".chart-panel[data-symbol]")) {
      return [node];
    }
    if (node.querySelectorAll) {
      return Array.prototype.slice.call(
        node.querySelectorAll(".chart-panel[data-symbol]"),
      );
    }
    return [];
  }

  document.body.addEventListener("htmx:load", function (evt) {
    findNewPanels(evt.detail.elt).forEach(function (panel) {
      if (panel._chartInitialized) {
        return;
      }
      panel._chartInitialized = true;
      initChart(panel);
    });
  });

  document.addEventListener("DOMContentLoaded", function () {
    var form = document.getElementById("subscribe-form");
    if (!form) {
      return;
    }
    form.addEventListener("submit", function (evt) {
      var symbol = form.symbol.value.trim().toUpperCase();
      var existing = document.getElementById("chart-panel-" + symbol);
      if (existing) {
        evt.preventDefault();
        evt.stopPropagation();
        existing.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    });
  });
})();
