// SPDX-License-Identifier: BSD-3-Clause
// Local progressive enhancement only. Semantic links/forms remain authoritative.
htmx.config.selfRequestsOnly = true;
htmx.config.allowEval = false;
htmx.config.allowScriptTags = false;
htmx.config.includeIndicatorStyles = false;
htmx.config.historyCacheSize = 0;
htmx.config.historyRestoreAsHxRequest = false;
document.addEventListener("htmx:beforeSwap", function (event) {
  if (event.detail.target.id === "workspace" && [400, 409, 503].includes(event.detail.xhr.status)) {
    event.detail.shouldSwap = true;
    event.detail.isError = false;
  }
});
// Status polling never replaces resource details or moves focus.
