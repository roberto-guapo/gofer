const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')

const source = fs.readFileSync(path.join(__dirname, '../assets/js/system-monitor.js'), 'utf8')
const messages = JSON.parse(fs.readFileSync(path.join(__dirname, '../internal/i18n/locales/en.json'), 'utf8'))
const events = {}, bodyEvents = {}, windowEvents = {}, timers = new Map()
let selectedInterval = null
let nextTimer = 0, enabled = false, calls = 0, pending, intersection, observed
function panel() {
  const nodes = {}
  return {
    hidden: true, isConnected: true, stale: false,
    querySelector(selector) { return nodes[selector] ||= { textContent: '', hidden: false, title: '' } },
    setAttribute(name) { if (name === 'data-stale') this.stale = true },
    removeAttribute(name) { if (name === 'data-stale') this.stale = false },
  }
}
let current = panel()
const document = {
  hidden: false, readyState: 'complete',
  body: { addEventListener(name, fn) { bodyEvents[name] = fn } },
  addEventListener(name, fn) { events[name] = fn },
  querySelector() { return current },
}
const context = {
  document, Number, AbortController,
  window: { addEventListener(name, fn) { windowEvents[name] = fn } },
  GoferSettings: { get(key) { return key === 'system_monitor_interval' ? selectedInterval : (enabled ? 'true' : 'false') }, set(key, value) { selectedInterval = value; bodyEvents['gofer:settings-changed']({detail:{key}}) } },
  goferT(key, params = {}) { return messages[key].replace(/\{(\w+)\}/g, (_, name) => params[name]) },
  setTimeout(fn, delay) { const id = ++nextTimer; timers.set(id, { fn, delay }); return id },
  clearTimeout(id) { timers.delete(id) },
  IntersectionObserver: class {
    constructor(fn) { intersection = fn }
    observe(node) { observed = node }
    disconnect() { observed = null }
  },
  fetch(url, options) {
    assert.equal(url, '/api/system/monitor')
    calls++
    return new Promise((resolve, reject) => {
      pending = { resolve, reject, signal: options.signal }
      options.signal.addEventListener('abort', () => reject(new Error('aborted')))
    })
  },
}
const tick = () => new Promise(resolve => setImmediate(resolve))
function runTimer(delay) {
  const entry = [...timers.entries()].find(([, timer]) => timer.delay === delay)
  assert.ok(entry, `missing ${delay}ms timer`)
  timers.delete(entry[0]); entry[1].fn()
}
function visible(value) { intersection([{ target: observed, isIntersecting: value }]) }
function setting(value) {
  enabled = value
  bodyEvents['gofer:settings-changed']({ detail: { key: 'system_monitor_enabled' } })
}

async function main() {
  vm.runInNewContext(source, context)
  assert.equal(calls, 0)
  assert.equal(timers.size, 0, 'disabled widget creates no polling timer')
  setting(true)
  assert.equal(current.hidden, false)
  assert.equal(timers.size, 0, 'offscreen widget does not poll')
  visible(true); runTimer(0)
  const first = pending
  events['htmx:afterSwap']()
  assert.equal(calls, 1, 'navigation must not duplicate an in-flight request')
  first.resolve({ ok: true, json: async () => ({ supported: true, cpu_percent: null, memory: { percent: 42, used: 1024, total: 2048 }, temperature_c: null, disk: null, uptime_seconds: 90061, process_memory_bytes: 104857600 }) })
  await tick()
  assert.equal(current.querySelector('[data-monitor-value="cpu"]').textContent, '—')
  assert.equal(current.querySelector('[data-monitor-value="memory"]').textContent, '42%')
  assert.equal(current.querySelector('[data-monitor-value="process"]').textContent, '100 MiB')
  assert.equal(current.querySelector('[data-monitor-value="uptime"]').textContent, '1d 1h 1m')
  assert.equal(timers.size, 1)
  document.hidden = true; events.visibilitychange()
  assert.equal(timers.size, 0, 'hidden browser tab stops refreshing')
  document.hidden = false; events.visibilitychange(); runTimer(0)
  pending.reject(new Error('offline')); await tick()
  assert.equal(current.stale, true)
  assert.match(current.querySelector('[data-monitor-status]').textContent, /unavailable/)
  runTimer(15000)
  const old = pending
  setting(false)
  assert.equal(old.signal.aborted, true)
  await tick()
  assert.equal(current.hidden, true)
  assert.equal(timers.size, 0, 'disabling cancels requests and retries')
  setting(true); visible(true); runTimer(0)
  const removed = current, previousRequest = pending
  removed.isConnected = false
  current = panel()
  events['htmx:afterSwap']()
  assert.equal(previousRequest.signal.aborted, true)
  await tick()
  visible(true); runTimer(0)
  pending.resolve({ status: 204, ok: true }); await tick()
  assert.match(current.querySelector('[data-monitor-status]').textContent, /disabled/)
  visible(false)
  assert.equal(timers.size, 0, 'closed mobile drawer stops refreshing')
  visible(true); runTimer(0)
  runTimer(5000); await tick()
  assert.equal(current.stale, true, 'request timeout shows unavailable state')
  for (const [raw, delay] of [['5', 5000], ['30', 30000], ['60', 60000], ['invalid', 15000], ['toString', 15000], ['15', 15000]]) {
    events.change({ target: { closest() { return { value: raw } } } })
    assert.equal(timers.size, 1, 'interval change replaces the previous timer')
    runTimer(delay)
    pending.resolve({ status: 204, ok: true }); await tick()
    assert.equal([...timers.values()][0].delay, delay, 'subsequent refresh uses chosen interval')
  }
  setting(false)
  events.change({ target: { closest() { return { value: '5' } } } })
  assert.equal(timers.size, 0, 'interval selection cannot start a disabled monitor')
  windowEvents.pagehide()
  assert.equal(timers.size, 0)
  console.log('System monitor: opt-in, visibility, navigation, aborts, timeout, stale state and formatting passed.')
}
main().catch(error => { console.error(error); process.exitCode = 1 })
