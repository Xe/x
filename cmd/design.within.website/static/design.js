(function () {
  function handleCopy(btn) {
    var pre = btn.parentElement && btn.parentElement.querySelector("pre");
    if (!pre) return;
    var text = pre.textContent || "";
    var orig = btn.dataset.origLabel || btn.textContent;
    btn.dataset.origLabel = orig;
    function ok() {
      btn.textContent = "Copied";
      btn.classList.add("copy-btn--ok");
      setTimeout(function () {
        btn.textContent = orig;
        btn.classList.remove("copy-btn--ok");
      }, 1400);
    }
    function fail() {
      btn.textContent = "Copy failed";
      setTimeout(function () {
        btn.textContent = orig;
      }, 1400);
    }
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(ok, fail);
      return;
    }
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.setAttribute("readonly", "");
    ta.style.position = "absolute";
    ta.style.left = "-9999px";
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand("copy");
      ok();
    } catch (err) {
      fail();
    }
    document.body.removeChild(ta);
  }

  function handleModalOpen(trigger) {
    var id = trigger.getAttribute("data-modal-open");
    if (!id) return;
    var dlg = document.getElementById(id);
    if (!dlg) return;
    if (typeof dlg.showModal === "function") {
      dlg.showModal();
    } else {
      dlg.setAttribute("open", "");
    }
  }

  document.addEventListener("click", function (e) {
    var copy = e.target.closest(".copy-btn");
    if (copy) {
      handleCopy(copy);
      return;
    }
    var trigger = e.target.closest("[data-modal-open]");
    if (trigger) {
      handleModalOpen(trigger);
      return;
    }
  });
})();
