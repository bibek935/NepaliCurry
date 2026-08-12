import React, { useState, useEffect } from 'react';

const API_BASE = window.location.origin.includes('-3000.')
  ? window.location.origin.replace('-3000.', '-8080.') // Codespaces環境用
  : `${window.location.protocol}//${window.location.hostname}:8080`; // AWS / ローカル環境用

function App() {
  const [tableNo, setTableNo] = useState(1);
  const [menu, setMenu] = useState([]);
  const [cart, setCart] = useState({});
  const [orders, setOrders] = useState([]);
  const [activeTab, setActiveTab] = useState('menu');

  const fetchOptions = (options = {}) => {
    return {
      ...options,
      credentials: 'include',
    };
  };

  useEffect(() => {
    fetchMenu();
    fetchOrders();
  }, []);

  const fetchMenu = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/menu`, fetchOptions());
      if (res.ok) {
        const data = await res.json();
        setMenu(data || []);
      }
    } catch (e) {
      console.error("メニュー取得エラー:", e);
    }
  };

  const fetchOrders = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/orders`, fetchOptions());
      if (res.ok) {
        const data = await res.json();
        setOrders(data || []);
      }
    } catch (e) {
      console.error("注文履歴取得エラー:", e);
    }
  };

  const updateCart = (itemId, delta) => {
    setCart((prev) => {
      const current = prev[itemId] || 0;
      const next = current + delta;
      if (next <= 0) {
        const copy = { ...prev };
        delete copy[itemId];
        return copy;
      }
      return { ...prev, [itemId]: next };
    });
  };

  const calculateCartTotal = () => {
    return Object.entries(cart).reduce((sum, [itemId, qty]) => {
      const item = menu.find((m) => m.id === Number(itemId));
      return sum + (item ? item.price * qty : 0);
    }, 0);
  };

  const handleOrderSubmit = async () => {
    const items = Object.entries(cart).map(([itemId, qty]) => ({
      menu_item_id: Number(itemId),
      quantity: qty,
    }));

    if (items.length === 0) return;

    try {
      const res = await fetch(`${API_BASE}/api/orders`, fetchOptions({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ table_no: Number(tableNo), items }),
      }));

      if (res.ok) {
        alert('ご注文を承りました！');
        setCart({});
        fetchOrders();
        setActiveTab('history');
      }
    } catch (e) {
      console.error("注文送信エラー:", e);
    }
  };

  const handlePay = async (orderId) => {
    if (!window.confirm('この注文の精算を完了しますか？')) return;
    try {
      const res = await fetch(`${API_BASE}/api/orders/pay`, fetchOptions({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_id: orderId }),
      }));

      if (res.ok) {
        alert('お支払いが完了しました。ありがとうございます！');
        fetchOrders();
      }
    } catch (e) {
      console.error("精算エラー:", e);
    }
  };

  return (
    <div className="app-container">
      <header className="navbar">
        <div className="nav-content">
          <h1 className="logo">Nepali Curry Restaurant</h1>
          <div className="table-selector">
            <label htmlFor="table-select">卓番号: </label>
            <input
              id="table-select"
              type="number"
              min="1"
              value={tableNo}
              onChange={(e) => setTableNo(Number(e.target.value))}
              style={{ width: '45px', padding: '4px', marginLeft: '6px', textAlign: 'center' }}
            />
          </div>
        </div>
      </header>

      <nav className="tab-bar">
        <button className={activeTab === 'menu' ? 'active' : ''} onClick={() => setActiveTab('menu')}>
          メニュー
        </button>
        <button className={activeTab === 'cart' ? 'active' : ''} onClick={() => setActiveTab('cart')}>
          注文確認 ({Object.values(cart).reduce((a, b) => a + b, 0)})
        </button>
        <button className={activeTab === 'history' ? 'active' : ''} onClick={() => { fetchOrders(); setActiveTab('history'); }}>
          注文履歴・精算
        </button>
      </nav>

      <main className="main-layout">
        {activeTab === 'menu' && (
          <div className="menu-list">
            {menu.map((item) => (
              <div key={item.id} className="menu-card card">
                <img src={item.image_url} alt={item.name} className="menu-img" />
                <div className="menu-details">
                  <div className="menu-header">
                    <h3>{item.name}</h3>
                    <span className="price">¥{item.price}</span>
                  </div>
                  <p className="description">{item.description}</p>
                  <div className="cart-controls">
                    <button onClick={() => updateCart(item.id, -1)} disabled={!cart[item.id]}>-</button>
                    <span>{cart[item.id] || 0}</span>
                    <button onClick={() => updateCart(item.id, 1)}>+</button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {activeTab === 'cart' && (
          <div className="cart-section card">
            <h2>カート内の注文確認</h2>
            {Object.keys(cart).length === 0 ? (
              <p className="empty-message">カートに商品が入っていません。</p>
            ) : (
              <div>
                <ul className="cart-list">
                  {Object.entries(cart).map(([itemId, qty]) => {
                    const item = menu.find((m) => m.id === Number(itemId));
                    if (!item) return null;
                    return (
                      <li key={itemId} className="cart-item">
                        <span>{item.name} x {qty}</span>
                        <span>¥{item.price * qty}</span>
                      </li>
                    );
                  })}
                </ul>
                <div className="cart-summary">
                  <h3>合計: ¥{calculateCartTotal()}</h3>
                  <button className="btn-primary" onClick={handleOrderSubmit}>注文を確定する</button>
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'history' && (
          <div className="history-section">
            <h2>注文履歴・精算</h2>
            {orders.length === 0 ? (
              <p className="empty-message">注文履歴がありません。</p>
            ) : (
              orders.map((order) => (
                <div key={order.id} className="order-card card">
                  <div className="order-header">
                    <span>注文 ID: #{order.id} (テーブル {order.table_no})</span>
                    <span className={`status ${order.status}`}>
                      {order.status === 'paid' ? '精算済み' : '未精算'}
                    </span>
                  </div>
                  <ul className="order-items">
                    {order.items.map((item) => (
                      <li key={item.id}>
                        {item.name} x {item.quantity} (¥{item.price * item.quantity})
                      </li>
                    ))}
                  </ul>
                  <div className="order-footer">
                    <span className="total">合計金額: ¥{order.total}</span>
                    {order.status === 'ordered' && (
                      <button className="btn-pay" onClick={() => handlePay(order.id)}>
                        お会計（精算）
                      </button>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </main>
    </div>
  );
}

export default App;