const form = document.querySelector('#search-form');
const usernameInput = document.querySelector('#username');
const status = document.querySelector('#status');
const clipsContainer = document.querySelector('#clips');
const loadMoreButton = document.querySelector('#load-more');
const periodSelect = document.querySelector('#period');
const dateFromInput = document.querySelector('#date-from');
const dateToInput = document.querySelector('#date-to');
const dateFromWrap = document.querySelector('#date-from-wrap');
const dateToWrap = document.querySelector('#date-to-wrap');
const minViewsInput = document.querySelector('#min-views');

let currentUsername = '';
let nextCursor = null;
let isLoading = false;

function formatDate(dateString) {
  return new Intl.DateTimeFormat('ru-RU', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(dateString));
}

function formatDuration(seconds) {
  const totalSeconds = Math.round(Number(seconds) || 0);
  const minutes = Math.floor(totalSeconds / 60);
  const remainingSeconds = String(totalSeconds % 60).padStart(2, '0');
  return `${minutes}:${remainingSeconds}`;
}

function getDateRange() {
  if (!periodSelect.value) return null;

  if (periodSelect.value === 'custom') {
    if (!dateFromInput.value || !dateToInput.value) {
      throw new Error('Для своего периода выберите обе даты.');
    }
    return {
      startedAt: new Date(`${dateFromInput.value}T00:00:00`).toISOString(),
      endedAt: new Date(`${dateToInput.value}T23:59:59.999`).toISOString(),
    };
  }

  const endedAt = new Date();
  const startedAt = new Date(endedAt);
  startedAt.setDate(startedAt.getDate() - Number(periodSelect.value));
  return { startedAt: startedAt.toISOString(), endedAt: endedAt.toISOString() };
}

function updateDateInputs() {
  const isCustom = periodSelect.value === 'custom';
  dateFromWrap.hidden = !isCustom;
  dateToWrap.hidden = !isCustom;
}

function clipCard(clip) {
  const article = document.createElement('article');
  article.className = 'clip-card';

  const image = document.createElement('img');
  image.src = clip.thumbnail_url;
  image.alt = `Превью: ${clip.title || 'Клип Twitch'}`;
  image.loading = 'lazy';

  const content = document.createElement('div');
  content.className = 'clip-content';

  const title = document.createElement('h2');
  title.textContent = clip.title || 'Без названия';

  const creator = document.createElement('p');
  creator.textContent = `Автор: ${clip.creator_name || 'Неизвестно'}`;

  const details = document.createElement('p');
  details.className = 'details';
  details.textContent = `${formatDate(clip.created_at)} · ${clip.view_count ?? 0} просмотров · ${formatDuration(clip.duration)}`;

  const link = document.createElement('a');
  link.href = clip.url;
  link.target = '_blank';
  link.rel = 'noopener noreferrer';
  link.textContent = 'Открыть на Twitch';

  const download = document.createElement('button');
  download.type = 'button';
  download.className = 'download-button';
  download.textContent = 'Скачать';
  download.addEventListener('click', async () => {
    download.disabled = true;
    const originalText = download.textContent;
    download.textContent = 'Скачиваем…';
    status.classList.remove('error');
    status.textContent = 'Открываем клип и готовим загрузку…';
    try {
      const response = await fetch(`/api/clips/${encodeURIComponent(clip.id)}/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: clip.url }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Не удалось скачать клип.');
      status.textContent = data.message;
    } catch (error) {
      status.classList.add('error');
      status.textContent = error.message;
    } finally {
      download.disabled = false;
      download.textContent = originalText;
    }
  });

  const actions = document.createElement('div');
  actions.className = 'clip-actions';
  actions.append(link, download);

  content.append(title, creator, details, actions);
  article.append(image, content);
  return article;
}

async function loadClips({ append = false } = {}) {
  if (isLoading || !currentUsername) return;

  isLoading = true;
  loadMoreButton.hidden = true;
  status.textContent = append ? 'Загружаем ещё клипы…' : 'Ищем клипы…';

  try {
    const params = new URLSearchParams({ username: currentUsername });
    const range = getDateRange();
    if (range) {
      params.set('started_at', range.startedAt);
      params.set('ended_at', range.endedAt);
    }
    params.set('min_views', minViewsInput.value || '0');
    if (append && nextCursor) params.set('after', nextCursor);

    const response = await fetch(`/api/clips?${params}`);
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Не удалось загрузить клипы.');

    if (!append) clipsContainer.replaceChildren();
    data.clips.forEach((clip) => clipsContainer.append(clipCard(clip)));
    nextCursor = data.pagination?.cursor || null;

    if (!data.clips.length && !append) {
      status.textContent = `У ${data.broadcaster.displayName} пока нет доступных клипов.`;
    } else {
      status.textContent = `Клипы ${data.broadcaster.displayName}: ${data.clips.length}${append ? ' добавлено' : ''}.`;
    }

    loadMoreButton.hidden = !nextCursor;
  } catch (error) {
    status.textContent = error.message;
    status.classList.add('error');
  } finally {
    isLoading = false;
  }
}

form.addEventListener('submit', (event) => {
  event.preventDefault();
  currentUsername = usernameInput.value.trim();
  nextCursor = null;
  status.classList.remove('error');
  loadClips();
});

periodSelect.addEventListener('change', updateDateInputs);

loadMoreButton.addEventListener('click', () => {
  status.classList.remove('error');
  loadClips({ append: true });
});
