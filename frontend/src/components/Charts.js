import React from 'react';
import { Line, Pie } from 'react-chartjs-2';
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Tooltip, Legend } from 'chart.js';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Tooltip, Legend);

function Charts({ stats, qps }) {
  const qpsChartData = {
    labels: Object.keys(qps),
    datasets: [{
      label: 'QPS',
      data: Object.values(qps),
      borderColor: 'rgba(75, 192, 192, 1)',
      fill: false,
    }],
  };

  const pieChartData = {
    labels: ['Разрешено', 'Заблокировано'],
    datasets: [{
      data: [stats.total_requests - stats.blocked, stats.blocked],
      backgroundColor: ['#36A2EB', '#FF6384'],
    }],
  };

  return (
    <div className="grid grid-cols-2 gap-4">
      <div>
        <h2 className="text-xl mb-2">QPS (Queries Per Second)</h2>
        <Line data={qpsChartData} />
      </div>
      <div>
        <h2 className="text-xl mb-2">Статистика блокировок</h2>
        <Pie data={pieChartData} />
      </div>
      <div className="col-span-2">
        <h2 className="text-xl mb-2">Топ доменов</h2>
        <ul className="list-disc pl-5">
          {Object.entries(stats.top_domains).map(([domain, count]) => (
            <li key={domain}>{domain}: {count}</li>
          ))}
        </ul>
      </div>
    </div>
  );
}

export default Charts;
