const newsContainer = document.getElementById('news-container');
const loadBtn = document.getElementById('load-news');
const countSpan = document.getElementById('new-count');

let lastUpdate = new Date().toISOString();

function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleString('ru-RU');
}

function fetchNews(limit = 10) {
    fetch(`/api/news/${limit}`)
        .then(res => res.json())
        .then(news => {
            newsContainer.innerHTML = '';
            news.forEach(item => {
                const div = document.createElement('div');
                div.className = 'news-item';
                div.innerHTML = `
                    <h3><a href="${item.link}" target="_blank">${item.title}</a></h3>
                    <p>${item.description || ''}</p>
                    <div class="source">
                        <span>${formatDate(item.date)}</span> | 
                        <a href="${item.feed}" target="_blank">Источник</a>
                    </div>`;
                newsContainer.appendChild(div);
            });
            lastUpdate = new Date().toISOString();
            countSpan.textContent = '0 новых';
            loadBtn.disabled = true;
        })
        .catch(err => console.error('Ошибка загрузки новостей:', err));
}

function fetchNewCount() {
    fetch(`/api/news/count?since=${encodeURIComponent(lastUpdate)}`)
        .then(res => res.json())
        .then(data => {
            const cnt = data.count || 0;
            countSpan.textContent = `${cnt} новых`;
            loadBtn.disabled = cnt === 0;
        })
        .catch(err => console.error('Ошибка получения количества новых:', err));
}

loadBtn.addEventListener('click', () => fetchNews());

fetchNews();
setInterval(fetchNewCount, 5 * 1000);
