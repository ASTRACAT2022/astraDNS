import React, { useState, useEffect } from 'react';
import Charts from './Charts';
import Settings from './Settings';

function Dashboard({ token, onLogout }) {
  const [stats, setStats] = useState({ total_requests: 0, blocked: 0, top_domains: {} });
  const [qps, setQps] = useState({});
  const [newDomain, setNewDomain] = useState('');
  const [tab, setTab] = useState('dashboard');

  const fetchData = async () => {
    try {
      const headers = { Authorization: token };
      const statsRes = await fetch('http://localhost:8080/api/stats', { headers });
      const statsData = await statsRes.json();
      setStats(statsData);

      const qpsRes = await fetch('http://localhost:8080/api/qps', { headers });
      const qpsData = await qpsRes.json();
      setQps(qpsData);
    } catch (err) {
      console.error('Ошибка загрузки данных:', err);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleBlockDomain = async (e) => {
    e.preventDefault();
    try {
      await fetch('http://localhost:8080/api/block', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token,
        },
        body: JSON.stringify({ domain: newDomain }),
      });
      setNewDomain('');
      fetchData();
    } catch (err) {
      console.error('Ошибка блокировки домена:', err);
    }
  };

  return (
    <div className="container mx-auto p-4">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">AstraDNS Dashboard</h1>
        <button onClick={onLogout} className="bg-red-500 text-white p-2 rounded hover:bg-red-600">
          Выйти
        </button>
      </div>
      <div className="flex mb-4">
        <button
          onClick={() => setTab('dashboard')}
          className={`px-4 py-2 mr-2 ${tab === 'dashboard' ? 'bg-blue-500 text-white' : 'bg-gray-200'}`}
        >
          Панель
        </button>
        <button
          onClick={() => setTab('settings')}
          className={`px-4 py-2 ${tab === 'settings' ? 'bg-blue-500 text-white' : 'bg-gray-200'}`}
        >
          Настройки
        </button>
      </div>
      {tab === 'dashboard' ? (
        <>
          <div className="mb-6">
            <h2 className="text-xl mb-2">Добавить домен в блоклист</h2>
            <form onSubmit={handleBlockDomain} className="flex">
              <input
                type="text"
                value={newDomain}
                onChange={(e) => setNewDomain(e.target.value)}
                placeholder="example.com"
                className="flex-grow p-2 border rounded-l"
              />
              <button type="submit" className="bg-blue-500 text-white p-2 rounded-r hover:bg-blue-600">
                Заблокировать
              </button>
            </form>
          </div>
          <Charts stats={stats} qps={qps} />
        </>
      ) : (
        <Settings token={token} />
      )}
    </div>
  );
}

export default Dashboard;
