(function () {
  'use strict';

  const MAX_FEED = 200;

  const badge    = document.getElementById('conn-badge');
  const counter  = document.getElementById('msg-counter');
  const feedList = document.getElementById('feed-list');
  const grid     = document.getElementById('channels-grid');

  let msgCount = 0;

  function wsURL() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    return proto + '://' + location.host + '/api/ws';
  }

  function setConnected(ok) {
    badge.textContent = ok ? 'connected' : 'disconnected';
    badge.className   = 'badge ' + (ok ? 'badge-connected' : 'badge-disconnected');
  }

  function upsertChannel(id, connected) {
    let card = document.getElementById('ch-' + id);
    if (!card) {
      card = document.createElement('div');
      card.id = 'ch-' + id;
      card.className = 'channel-card';
      card.innerHTML = '<div class="ch-name">' + id + '</div><div class="ch-badge"></div>';
      grid.appendChild(card);
    }
    const b = card.querySelector('.ch-badge');
    b.textContent  = connected ? 'connected' : 'disconnected';
    b.className    = 'ch-badge badge ' + (connected ? 'badge-connected' : 'badge-disconnected');
  }

  function prependFeed(ev) {
    const ts   = ev.timestamp ? ev.timestamp.substring(11, 19) : '';
    const li   = document.createElement('li');
    const typeClass = (ev.type === 'message.sent') ? 'feed-type sent' : 'feed-type';
    const content   = ev.content ? ev.content.substring(0, 120) : '';
    li.innerHTML =
      '<span class="feed-ts">' + ts + '</span>' +
      '<span class="feed-ch">[' + (ev.channel_id || '') + ']</span>' +
      '<span class="' + typeClass + '">' + ev.type + '</span>' +
      content;
    feedList.insertBefore(li, feedList.firstChild);

    // Trim to MAX_FEED items
    while (feedList.children.length > MAX_FEED) {
      feedList.removeChild(feedList.lastChild);
    }
  }

  function connect() {
    const ws = new WebSocket(wsURL());

    ws.onopen = function () { setConnected(true); };

    ws.onmessage = function (e) {
      var ev;
      try { ev = JSON.parse(e.data); } catch (_) { return; }

      switch (ev.type) {
        case 'message.received':
        case 'message.sent':
          msgCount++;
          counter.textContent = msgCount;
          prependFeed(ev);
          break;
        case 'channel.connected':
          upsertChannel(ev.channel_id, true);
          break;
        case 'channel.disconnected':
          upsertChannel(ev.channel_id, false);
          break;
        default:
          prependFeed(ev);
      }
    };

    ws.onclose = function () {
      setConnected(false);
      setTimeout(connect, 3000);
    };

    ws.onerror = function () { ws.close(); };
  }

  connect();
}());
