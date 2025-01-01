import React, { useState } from 'react';

function App() {
  const [response, setResponse] = useState(null);
  const [error, setError] = useState(null);

  const testCORS = async () => {
    try {
      const res = await fetch('http://localhost:8080', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      const data = await res.json();
      setResponse(data);
      setError(null);
    } catch (err) {
      setError(err.message);
      setResponse(null);
    }
  };

  return (
      <div style={{ textAlign: 'center', padding: '20px' }}>
        <h1>CORSテスト</h1>
        <button onClick={testCORS}>CORS APIを呼び出す</button>
        {response && <div><h2>レスポンス:</h2><pre>{JSON.stringify(response, null, 2)}</pre></div>}
        {error && <div><h2>エラー:</h2><pre>{error}</pre></div>}
      </div>
  );
}

export default App;
