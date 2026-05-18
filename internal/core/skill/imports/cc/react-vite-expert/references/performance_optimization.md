# React 性能优化指南

优化 React + Vite 应用程序以获得最佳性能的综合指南。

## 要跟踪的性能指标

### 核心网络生命力
- **LCP（最大内容油漆）**：< 2.5 秒（良好）
- **FID（首次输入延迟）**：< 100ms（良好）
- **CLS（累积布局偏移）**：< 0.1（良好）
- **FCP（首次内容绘制）**：< 1.8s（良好）
- **TTI（交互时间）**：< 3.8 秒（良好）

### 捆绑包大小目标
- **初始 JS 包**：< 200KB（gzip 压缩）
- **总 JS**：< 500KB（gzip 压缩）
- **CSS**：< 50KB（压缩）
- **图像**：使用WebP/AVIF，延迟加载

## React 渲染优化

### 1. React.memo() - 防止不必要的重新渲染

```typescript
// ❌ Bad: Re-renders on every parent render
export const ExpensiveComponent = ({ data }) => {
  return <div>{/* Complex rendering */}</div>;
};

// ✅ Good: Only re-renders when props change
export const ExpensiveComponent = React.memo(({ data }) => {
  return <div>{/* Complex rendering */}</div>;
});

// ✅ Best: Custom comparison for complex props
export const ExpensiveComponent = React.memo(
  ({ data }) => {
    return <div>{/* Complex rendering */}</div>;
  },
  (prevProps, nextProps) => {
    // Return true if props are equal (don't re-render)
    return prevProps.data.id === nextProps.data.id;
  }
);
```

**何时使用 React.memo():**
- ✅ 经常渲染的纯组件
- ✅ 渲染成本高昂的组件
- ✅ 经常接收相同 props 的组件
- ❌ 简单的组件（开销 > 收益）
- ❌ 很少重新渲染的组件

### 2. useMemo() - 记忆昂贵的计算

```typescript
// ❌ Bad: Recalculates on every render
function ProductList({ products }) {
  const sortedProducts = products.sort((a, b) => b.price - a.price);
  const filteredProducts = sortedProducts.filter(p => p.inStock);

  return <div>{/* Render filtered products */}</div>;
}

// ✅ Good: Only recalculates when products change
function ProductList({ products }) {
  const processedProducts = useMemo(() => {
    const sorted = [...products].sort((a, b) => b.price - a.price);
    return sorted.filter(p => p.inStock);
  }, [products]);

  return <div>{/* Render processed products */}</div>;
}
```

**何时使用 useMemo():**
- ✅ 昂贵的计算（排序、过滤大数组）
- ✅ 创建作为道具传递给记忆组件的对象/数组
- ✅ 复杂的转变
- ❌简单的计算（开销>收益）

### 3. useCallback() - 记忆函数

```typescript
// ❌ Bad: Creates new function on every render (breaks memo)
function Parent() {
  const handleClick = (id) => {
    console.log(id);
  };

  return <MemoizedChild onClick={handleClick} />;
}

// ✅ Good: Function identity stays same
function Parent() {
  const handleClick = useCallback((id) => {
    console.log(id);
  }, []); // Empty deps: function never changes

  return <MemoizedChild onClick={handleClick} />;
}

// ✅ With dependencies
function Parent() {
  const [userId, setUserId] = useState('123');

  const handleClick = useCallback((id) => {
    console.log(userId, id);
  }, [userId]); // Re-creates when userId changes

  return <MemoizedChild onClick={handleClick} />;
}
```

**何时使用 useCallback():**
- ✅ 将回调传递给记忆的子组件
- ✅ 其他钩子的依赖数组中的回调
- ✅ 事件处理程序传递给许多孩子
- ❌ 简单的事件处理程序不影响记忆

### 4. 使用 React.lazy() 进行代码分割

```typescript
// ❌ Bad: Loads everything upfront
import Dashboard from './pages/Dashboard';
import Profile from './pages/Profile';
import Settings from './pages/Settings';

// ✅ Good: Loads on-demand
const Dashboard = lazy(() => import('./pages/Dashboard'));
const Profile = lazy(() => import('./pages/Profile'));
const Settings = lazy(() => import('./pages/Settings'));

// Usage with Suspense
function App() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <Routes>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/profile" element={<Profile />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Suspense>
  );
}
```

**基于路由的代码分割模式：**
```typescript
// pages/DashboardPage/DashboardPage.lazy.tsx
import { lazy } from 'react';

export const DashboardPageLazy = lazy(() =>
  import('./DashboardPage').then(module => ({
    default: module.DashboardPage
  }))
);
```

### 5. 长列表的虚拟化

```typescript
// ❌ Bad: Renders 10,000 items (performance killer)
function ProductList({ products }) {
  return (
    <div>
      {products.map(product => (
        <ProductCard key={product.id} product={product} />
      ))}
    </div>
  );
}

// ✅ Good: Only renders visible items
import { FixedSizeList } from 'react-window';

function ProductList({ products }) {
  return (
    <FixedSizeList
      height={600}
      itemCount={products.length}
      itemSize={100}
      width="100%"
    >
      {({ index, style }) => (
        <div style={style}>
          <ProductCard product={products[index]} />
        </div>
      )}
    </FixedSizeList>
  );
}
```

**图书馆：**
- **react-window**：更轻，大多数用例
- **react-virtualized**：更多功能，更重

### 6. 去抖动和节流

```typescript
// Custom debounce hook
function useDebounce<T>(value: T, delay: number = 500): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => clearTimeout(handler);
  }, [value, delay]);

  return debouncedValue;
}

// Usage in search
function SearchComponent() {
  const [searchTerm, setSearchTerm] = useState('');
  const debouncedSearchTerm = useDebounce(searchTerm, 300);

  useEffect(() => {
    if (debouncedSearchTerm) {
      // API call only after 300ms of no typing
      fetchSearchResults(debouncedSearchTerm);
    }
  }, [debouncedSearchTerm]);

  return (
    <input
      value={searchTerm}
      onChange={(e) => setSearchTerm(e.target.value)}
    />
  );
}
```

### 7.避免内联对象和函数

```typescript
// ❌ Bad: Creates new object/function every render
function Component() {
  return (
    <>
      <Child style={{ padding: 10 }} />
      <Child onClick={() => console.log('click')} />
    </>
  );
}

// ✅ Good: Define outside or use constants
const CHILD_STYLE = { padding: 10 };

function Component() {
  const handleClick = () => console.log('click');

  return (
    <>
      <Child style={CHILD_STYLE} />
      <Child onClick={handleClick} />
    </>
  );
}
```

## Vite构建优化

### vite.config.ts - 生产优化

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';

export default defineConfig({
  plugins: [
    react(),
    // Bundle analyzer
    visualizer({
      filename: './dist/stats.html',
      open: true,
      gzipSize: true,
      brotliSize: true,
    }),
  ],

  build: {
    // Target modern browsers
    target: 'esnext',

    // Chunk size warnings
    chunkSizeWarningLimit: 500,

    // Minification
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true, // Remove console.log in production
        drop_debugger: true,
      },
    },

    // Rollup options
    rollupOptions: {
      output: {
        // Manual chunks for better caching
        manualChunks: {
          // Vendor chunk for large libraries
          vendor: ['react', 'react-dom', 'react-router-dom'],
          // UI components chunk
          ui: ['@/components/ui'],
        },

        // Asset file naming
        assetFileNames: (assetInfo) => {
          const info = assetInfo.name.split('.');
          let extType = info[info.length - 1];
          if (/png|jpe?g|svg|gif|tiff|bmp|ico/i.test(extType)) {
            extType = 'images';
          } else if (/woff|woff2|ttf|otf|eot/i.test(extType)) {
            extType = 'fonts';
          }
          return `assets/${extType}/[name]-[hash][extname]`;
        },
        chunkFileNames: 'assets/js/[name]-[hash].js',
        entryFileNames: 'assets/js/[name]-[hash].js',
      },
    },

    // Source maps for production debugging (optional)
    sourcemap: false,

    // CSS code splitting
    cssCodeSplit: true,
  },

  // Optimize deps
  optimizeDeps: {
    include: ['react', 'react-dom'],
  },
});
```

### 手动分块策略

```typescript
// Separate chunks by route
manualChunks: (id) => {
  // Vendor chunk
  if (id.includes('node_modules')) {
    if (id.includes('react') || id.includes('react-dom')) {
      return 'react-vendor';
    }
    if (id.includes('@tanstack/react-query')) {
      return 'query-vendor';
    }
    return 'vendor';
  }

  // Feature-based chunks
  if (id.includes('/src/features/auth')) {
    return 'auth';
  }
  if (id.includes('/src/features/dashboard')) {
    return 'dashboard';
  }
}
```

## 图像优化

### 1. 现代格式 (WebP/AVIF)

```typescript
// Use picture element for format fallback
<picture>
  <source srcSet="image.avif" type="image/avif" />
  <source srcSet="image.webp" type="image/webp" />
  <img src="image.jpg" alt="Description" loading="lazy" />
</picture>
```

### 2. 延迟加载

```typescript
// Native lazy loading
<img src="image.jpg" loading="lazy" alt="Description" />

// Intersection Observer for custom lazy loading
function LazyImage({ src, alt }: { src: string; alt: string }) {
  const [imageSrc, setImageSrc] = useState<string | null>(null);
  const imgRef = useRef<HTMLImageElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) {
        setImageSrc(src);
        observer.disconnect();
      }
    });

    if (imgRef.current) {
      observer.observe(imgRef.current);
    }

    return () => observer.disconnect();
  }, [src]);

  return (
    <img
      ref={imgRef}
      src={imageSrc || undefined}
      alt={alt}
      style={{ minHeight: '200px' }} // Prevent layout shift
    />
  );
}
```

### 3. 响应式图像

```typescript
<img
  src="image-800w.jpg"
  srcSet="
    image-400w.jpg 400w,
    image-800w.jpg 800w,
    image-1200w.jpg 1200w
  "
  sizes="
    (max-width: 600px) 400px,
    (max-width: 1200px) 800px,
    1200px
  "
  alt="Description"
  loading="lazy"
/>
```

## 网络性能

### 1. API请求优化

```typescript
// ❌ Bad: Multiple sequential requests
async function loadUserData(userId: string) {
  const user = await fetchUser(userId);
  const posts = await fetchPosts(userId);
  const comments = await fetchComments(userId);
  return { user, posts, comments };
}

// ✅ Good: Parallel requests
async function loadUserData(userId: string) {
  const [user, posts, comments] = await Promise.all([
    fetchUser(userId),
    fetchPosts(userId),
    fetchComments(userId),
  ]);
  return { user, posts, comments };
}

// ✅ Best: Use React Query for automatic optimization
function useUserData(userId: string) {
  const user = useQuery(['user', userId], () => fetchUser(userId));
  const posts = useQuery(['posts', userId], () => fetchPosts(userId));
  const comments = useQuery(['comments', userId], () => fetchComments(userId));

  return { user, posts, comments };
}
```

### 2. 请求重复数据删除

```typescript
// React Query automatically deduplicates identical requests
function Component1() {
  const { data } = useQuery(['user', '123'], fetchUser);
  // ...
}

function Component2() {
  const { data } = useQuery(['user', '123'], fetchUser);
  // Only one actual API call made!
}
```

### 3. 预取

```typescript
import { queryClient } from '@/lib/queryClient';

function ProductList() {
  const handleMouseEnter = (productId: string) => {
    // Prefetch product details before user clicks
    queryClient.prefetchQuery(['product', productId], () =>
      fetchProduct(productId)
    );
  };

  return (
    <div>
      {products.map(product => (
        <div
          key={product.id}
          onMouseEnter={() => handleMouseEnter(product.id)}
        >
          {product.name}
        </div>
      ))}
    </div>
  );
}
```

## CSS 性能

### 1.CSS模块（推荐）

```typescript
// Button.module.css
.button {
  padding: 10px 20px;
  border: none;
  border-radius: 4px;
}

.primary {
  background: blue;
  color: white;
}

// Button.tsx
import styles from './Button.module.css';

export const Button = ({ variant = 'primary', children }) => (
  <button className={`${styles.button} ${styles[variant]}`}>
    {children}
  </button>
);
```

**好处：**
- ✅ 范围样式（无冲突）
- ✅ 可摇树
- ✅ 更好的代码分割
- ✅ 较小的捆绑包

### 2. 避免运行时 CSS-in-JS

```typescript
// ❌ Slow: Runtime CSS-in-JS (styled-components, emotion)
const Button = styled.button`
  padding: 10px 20px;
  background: ${props => props.primary ? 'blue' : 'gray'};
`;

// ✅ Fast: CSS Modules or Vanilla Extract (zero-runtime)
import styles from './Button.module.css';

export const Button = ({ primary, children }) => (
  <button className={primary ? styles.primary : styles.secondary}>
    {children}
  </button>
);
```

### 3. 关键的 CSS 内联

```typescript
// vite.config.ts - Inline critical CSS
import { defineConfig } from 'vite';

export default defineConfig({
  build: {
    cssCodeSplit: true,
    assetsInlineLimit: 4096, // Inline assets < 4KB
  },
});
```

## 状态管理绩效

### Zustand 与选择器（防止重新渲染）

```typescript
// ❌ Bad: Component re-renders on ANY store change
function Component() {
  const store = useStore(); // Gets entire store
  return <div>{store.user.name}</div>;
}

// ✅ Good: Only re-renders when user.name changes
function Component() {
  const userName = useStore(state => state.user.name); // Selector
  return <div>{userName}</div>;
}

// ✅ Best: Shallow equality for objects
function Component() {
  const user = useStore(
    state => ({ name: state.user.name, email: state.user.email }),
    shallow // Only re-render if object values change
  );
  return <div>{user.name} - {user.email}</div>;
}
```

### 带重新选择的 Redux（记忆选择器）

```typescript
import { createSelector } from 'reselect';

// Input selectors
const selectProducts = (state) => state.products;
const selectFilter = (state) => state.filter;

// Memoized selector (only recalculates when inputs change)
const selectFilteredProducts = createSelector(
  [selectProducts, selectFilter],
  (products, filter) => {
    return products.filter(p => p.category === filter);
  }
);

// Usage
const filteredProducts = useSelector(selectFilteredProducts);
```

## 性能监控

### 网络生命体征追踪

```typescript
// src/lib/webVitals.ts
import { onCLS, onFID, onFCP, onLCP, onTTFB } from 'web-vitals';

function sendToAnalytics({ name, delta, value, id }) {
  // Send to your analytics service
  console.log(name, delta, value, id);
}

export function reportWebVitals() {
  onCLS(sendToAnalytics);
  onFID(sendToAnalytics);
  onFCP(sendToAnalytics);
  onLCP(sendToAnalytics);
  onTTFB(sendToAnalytics);
}

// main.tsx
import { reportWebVitals } from './lib/webVitals';

reportWebVitals();
```

### React DevTools 分析器

```typescript
import { Profiler } from 'react';

function onRenderCallback(
  id, // Component ID
  phase, // "mount" or "update"
  actualDuration, // Time spent rendering
  baseDuration, // Estimated time without memoization
  startTime,
  commitTime
) {
  console.log({ id, phase, actualDuration, baseDuration });
}

// Wrap component to profile
<Profiler id="Dashboard" onRender={onRenderCallback}>
  <Dashboard />
</Profiler>
```

## 绩效检查表

### 发展
- [ ] 使用 React DevTools Profiler 识别慢速组件
- [ ] 检查是否有不必要的重新渲染
- [ ] 验证记忆功能是否正常工作
- [ ] 在开发过程中监控包大小

### 生产前
- [ ] 运行捆绑分析器（`npm run build`带可视化仪）
- [ ] 检查包大小 < 目标
- [ ] 验证代码分割是否有效
- [ ] 慢速 3G 连接测试
- [ ] 在低端设备上测试
- [ ] 运行 Lighthouse 审核（分数 > 90）
- [ ] 衡量核心网络生命力
- [ ] 启用压缩 (gzip/brotli)
- [ ] 为资产配置 CDN
- [ ] 删除console.log语句
- [ ] 启用缩小

### 生产监控
- [ ] 跟踪生产中的 Web Vitals
- [ ] 监控每个部署的包大小
- [ ] 设置绩效预算
- [ ] 创建回归警报
