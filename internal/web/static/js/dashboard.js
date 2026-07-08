(function () {
  var CHART_HEIGHT = 480;

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
    overlay.className = "spinner-overlay";
    var spinner = document.createElement("div");
    spinner.className = "spinner";
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
    alert.className = "alert alert--danger";
    alert.textContent =
      panel.dataset.symbol +
      " " +
      panel.dataset.timeframe +
      ": " +
      message +
      " ";

    var dismiss = document.createElement("button");
    dismiss.type = "button";
    dismiss.className = "btn";
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
    var tf = panel.dataset.timeframe;
    var es = new EventSource(
      "/web/events?symbol=" +
        encodeURIComponent(symbol) +
        "&timeframe=" +
        encodeURIComponent(tf),
    );
    es.addEventListener("bar", function (evt) {
      panel._series.update(JSON.parse(evt.data));
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
      upColor: themeColor("--color-green-600", "#16a34a"),
      downColor: themeColor("--color-red-600", "#dc2626"),
      borderVisible: false,
      wickUpColor: themeColor("--color-green-600", "#16a34a"),
      wickDownColor: themeColor("--color-red-600", "#dc2626"),
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
      var tf = form.timeframe.value;
      var existing = document.getElementById(
        "chart-panel-" + symbol + "-" + tf,
      );
      if (existing) {
        evt.preventDefault();
        evt.stopPropagation();
        existing.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    });
  });
})();
