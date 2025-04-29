import React, { useState, useEffect } from 'react';

function Settings({ token }) {
  const [config, setConfig] = useState({ dns_port: '', api_port: '', jwt_secret: '' });
  const [blocklists, setBlocklists] = useState([]);
  const [redirects, setRedirects] = useState([]);
  const [newBlocklist, setNewBlocklist] = useState('');
  const [newRedirect, setNewRedirect] = useState({ domain: '', dns: '' });

  const fetchData = async () => {
    try {
      const headers = { Authorization: token };
      const configRes = await fetch('http://localhost:8080/api/config', { headers });
      const configData = await configRes.json();
      setConfig(configData);

      const blocklistsRes = await fetch('http://localhost:8080/api/blocklists', { headers });
      const blocklistsData = await blocklistsRes.json();
      setBlocklists(blocklistsData);

      const redirectsRes = await fetch('http://localhost:8080/api/redirects', { headers });
      const redirectsData = await redirectsRes.json();
      setRedirects(redirectsData);
    } catch (err) {
      console.error('Ошибка загрузки настроек:', err);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleConfigUpdate = async (e) => {
    e.preventDefault();
    try {
      await fetch('http://localhost:8080/api/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token,
        },
        body: JSON.stringify(config),
      });
      alert('Конфигурация обновлена');
    } catch (err) {
      console.error('Ошибка обновления конфигурации:', err);
    }
  };

  const handleAddBlocklist = async (e) => {
    e.preventDefault();
    try {
      await fetch('http://localhost:8080/api/blocklists', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token,
        },
        body: JSON.stringify({ url: newBlocklist }),
      });
      setNewBlocklist('');
      fetchData();
    } catch (err) {
      console.error('Ошибка добавления блоклиста:', err);
    }
  };

  const handleDeleteBlocklist = async (id) => {
    try {
      await fetch(`http://localhost:8080/api/blocklists/${id}`, {
        method: 'DELETE',
        headers: { Authorization: token },
      });
      fetchData();
    } catch (err) {
      console.error('Ошибка удаления блоклиста:', err);
    }
  };

  const handleAddRedirect = async (e) => {
    e.preventDefault();
    try {
      await fetch('http://localhost:8080/api/redirects', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token,
        },
        body: JSON.stringify(newRedirect),
      });
      setNewRedirect({ domain: '', dns: '' });
      fetchData();
    } catch (err) {
      console.error('Ошибка добавления перенаправления:', err);
    }
  };

  const handleDeleteRedirect = async (id) => {
    try {
      await fetch(`http://localhost:8080/api/redirects/${id}`, {
        method: 'DELETE',
        headers: { Authorization: token },
      });
      fetchData();
    } catch (err) {
      console.error('Ошибка удаления перенаправления:', err);
    }
  };

  return (
    <div>
      <h2 className="text-xl mb-4">Настройки</h2>

      <div className="mb-6">
        <h3 className="text-lg mb-2">Конфигурация</h3>
        <form onSubmit={handleConfigUpdate}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">DNS порт</label>
            <input
              type="number"
              value={config.dns_port}
              onChange={(e) => setConfig({ ...config, dns_port: e.target.value })}
              className="w-full p-2 border rounded"
            />
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">API порт</label>
            <input
              type="number"
              value={config.api_port}
              onChange={(e) => setConfig({ ...config, api_port: e.target.value })}
              className="w-full p-2 border rounded"
            />
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">JWT секрет</label>
            <input
              type="text"
              value={config.jwt_secret}
              onChange={(e) => setConfig({ ...config, jwt_secret: e.target.value })}
              className="w-full p-2 border rounded"
            />
          </div>
          <button type="submit" className="bg-blue-500 text-white p-2 rounded hover:bg-blue-600">
            Сохранить
          </button>
        </form>
      </div>

      <div className="mb-6">
        <h3 className="text-lg mb-2">Блоклисты</h3>
        <form onSubmit={handleAddBlocklist} className="flex mb-4">
          <input
            type="text"
            value={newBlocklist}
            onChange={(e) => setNewBlocklist(e.target.value)}
            placeholder="Путь к файлу блоклиста"
            className="flex-grow p-2 border rounded-l"
          />
          <button type="submit" className="bg-blue-500 text-white p-2 rounded-r hover:bg-blue-600">
            Добавить
          </button>
        </form>
        <ul className="list-disc pl-5">
          {blocklists.map((bl) => (
            <li key={bl.id} className="flex justify-between">
              <span>{bl.url}</span>
              <button
                onClick={() => handleDeleteBlocklist(bl.id)}
                className="text-red-500 hover:text-red-700"
              >
                Удалить
              </button>
            </li>
          ))}
        </ul>
      </div>

      <div>
        <h3 className="text-lg mb-2">Перенаправления</h3>
        <form onSubmit={handleAddRedirect} className="flex mb-4">
          <input
            type="text"
            value={newRedirect.domain}
            onChange={(e) => setNewRedirect({ ...newRedirect, domain: e.target.value })}
            placeholder="Домен (example.com)"
            className="flex-grow p-2 border rounded-l"
          />
          <input
            type="text"
            value={newRedirect.dns}
            onChange={(e) => setNewRedirect({ ...newRedirect, dns: e.target.value })}
            placeholder="DNS (1.1.1.1:853)"
            className="flex-grow p-2 border"
          />
          <button type="submit" className="bg-blue-500 text-white p-2 rounded-r hover:bg-blue-600">
            Добавить
          </button>
        </form>
        <ul className="list-disc pl-5">
          {redirects.map((rd) => (
            <li key={rd.id} className="flex justify-between">
              <span>{rd.domain} → {rd.dns}</span>
              <button
                onClick={() => handleDeleteRedirect(rd.id)}
                className="text-red-500 hover:text-red-700"
              >
                Удалить
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

export default Settings;
