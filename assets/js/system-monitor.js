// One polling loop for the visible sidebar, with no worker when disabled.
;(function () {
  var key = 'system_monitor_enabled'
  var intervalKey = 'system_monitor_interval'
  function interval() {
    var raw = typeof GoferSettings !== 'undefined' && GoferSettings.get(intervalKey)
    return raw === '5' || raw === '15' || raw === '30' || raw === '60' ? Number(raw) * 1000 : 15000
  }
  var target = null
  var observer = null
  var visible = false
  var timer = null
  var controller = null
  var generation = 0

  function enabled() {
    return typeof GoferSettings !== 'undefined' && GoferSettings.get(key) === 'true'
  }
  function stop() {
    generation++
    if (timer !== null) clearTimeout(timer)
    timer = null
    if (controller) controller.abort()
    controller = null
  }
  function active() {
    return target && target.isConnected && !target.hidden && visible && !document.hidden && enabled()
  }
  function status(message) {
    var node = target.querySelector('[data-monitor-status]')
    node.textContent = message
    node.hidden = !message
  }
  function value(name, text, title) {
    var node = target.querySelector('[data-monitor-value="' + name + '"]')
    node.textContent = text
    node.title = title || ''
  }
  function number(n) { return typeof n === 'number' && Number.isFinite(n) }
  function percent(n) { return number(n) ? Math.round(n) + '%' : '—' }
  function bytes(n) {
    if (!number(n)) return '—'
    return n >= 1073741824 ? (n / 1073741824).toFixed(1) + ' GiB' : Math.round(n / 1048576) + ' MiB'
  }
  function usageTitle(u) {
    return u ? goferT('js.monitor.used', { used: bytes(u.used), total: bytes(u.total) }) : ''
  }
  function render(data) {
    target.removeAttribute('data-stale')
    value('cpu', percent(data.cpu_percent), data.cpu_percent === null ? goferT('js.monitor.sampling') : '')
    value('memory', percent(data.memory && data.memory.percent), usageTitle(data.memory))
    value('temperature', number(data.temperature_c) ? Math.round(data.temperature_c) + ' °C' : '—')
    value('disk', percent(data.disk && data.disk.percent), usageTitle(data.disk))
    value('process', bytes(data.process_memory_bytes))
    var seconds = data.uptime_seconds
    value('uptime', number(seconds) ? goferT('js.monitor.uptime', {
      days: Math.floor(seconds / 86400), hours: Math.floor(seconds / 3600) % 24, minutes: Math.floor(seconds / 60) % 60,
    }) : '—')
    status(data.supported ? '' : goferT('js.monitor.unavailable'))
  }
  function schedule(delay) {
    if (active() && timer === null && !controller) timer = setTimeout(poll, delay)
  }
  function poll() {
    timer = null
    if (!active() || controller) return
    var ownGeneration = generation
    var request = new AbortController()
    controller = request
    var timeout = setTimeout(function () { request.abort() }, 5000)
    fetch('/api/system/monitor', { signal: request.signal, cache: 'no-store', headers: { Accept: 'application/json' } })
      .then(function (response) {
        if (response.status === 204) return null // Preference save may still be in flight.
        if (!response.ok) throw new Error('monitor unavailable')
        return response.json()
      })
      .then(function (data) {
        if (ownGeneration !== generation || !active()) return
        if (data) render(data)
        else {
          target.setAttribute('data-stale', '')
          status(goferT('js.monitor.disabled'))
        }
      })
      .catch(function () {
        if (ownGeneration !== generation || !active()) return
        target.setAttribute('data-stale', '')
        status(goferT('js.monitor.unavailable'))
      })
      .finally(function () {
        clearTimeout(timeout)
        if (ownGeneration !== generation) return
        controller = null
        schedule(interval())
      })
  }
  function reconcile() {
    var next = document.querySelector('[data-system-monitor]')
    if (next !== target) {
      stop()
      if (observer) observer.disconnect()
      observer = null
      target = next
      visible = false
    }
    if (!target) return
    target.hidden = !enabled()
    if (target.hidden) {
      stop()
      if (observer) observer.disconnect()
      observer = null
      visible = false
      return
    }
    if (!observer) {
      observer = new IntersectionObserver(function (entries) {
        visible = entries.some(function (entry) { return entry.target === target && entry.isIntersecting })
        if (active()) schedule(0)
        else stop()
      })
      observer.observe(target)
    }
    if (active()) schedule(0)
  }
  document.body.addEventListener('gofer:settings-changed', function (event) {
    if (event.detail.key === key) reconcile()
    if (event.detail.key === intervalKey) {
      stop()
      schedule(interval())
    }
  })
  document.addEventListener('change', function (event) {
    var input = event.target.closest('[data-system-monitor-interval]')
    if (input && typeof GoferSettings !== 'undefined') GoferSettings.set(intervalKey, input.value)
  })
  document.addEventListener('visibilitychange', function () {
    if (document.hidden) stop()
    else reconcile()
  })
  window.addEventListener('pagehide', stop)
  window.addEventListener('pageshow', reconcile)
  document.addEventListener('htmx:afterSwap', reconcile)
  if (window.goferOnContent) window.goferOnContent('[data-system-monitor]', reconcile)
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', reconcile)
  else reconcile()
})()
