# 最优 React + Vite 项目结构示例

这是一个完整的、可用于生产的项目结构示例。

## 目录树

```
my-react-app/
├── .github/
│   └── workflows/
│       └── ci.yml                   # GitHub Actions CI/CD
│
├── .husky/
│   ├── pre-commit                   # Lint staged files
│   └── pre-push                     # Run tests before push
│
├── public/
│   ├── favicon.ico
│   └── robots.txt
│
├── src/
│   ├── app/
│   │   ├── App.tsx                  # Root component
│   │   ├── App.test.tsx
│   │   ├── router.tsx               # Route configuration
│   │   └── providers.tsx            # Context providers wrapper
│   │
│   ├── features/                    # Feature modules
│   │   ├── auth/
│   │   │   ├── components/
│   │   │   │   ├── LoginForm/
│   │   │   │   │   ├── LoginForm.tsx
│   │   │   │   │   ├── LoginForm.types.ts
│   │   │   │   │   ├── LoginForm.module.css
│   │   │   │   │   ├── LoginForm.test.tsx
│   │   │   │   │   └── index.ts
│   │   │   │   └── RegisterForm/
│   │   │   ├── hooks/
│   │   │   │   ├── useAuth.ts
│   │   │   │   └── useLogin.ts
│   │   │   ├── api/
│   │   │   │   ├── authApi.ts
│   │   │   │   └── authApi.types.ts
│   │   │   └── index.ts
│   │   │
│   │   └── dashboard/
│   │       ├── components/
│   │       ├── hooks/
│   │       └── index.ts
│   │
│   ├── components/                  # Shared components
│   │   ├── ui/
│   │   │   ├── Button/
│   │   │   ├── Input/
│   │   │   ├── Modal/
│   │   │   └── index.ts
│   │   ├── layout/
│   │   │   ├── Header/
│   │   │   ├── Footer/
│   │   │   └── index.ts
│   │   └── form/
│   │       └── FormField/
│   │
│   ├── hooks/                       # Shared custom hooks
│   │   ├── useDebounce.ts
│   │   ├── useLocalStorage.ts
│   │   ├── useMediaQuery.ts
│   │   └── index.ts
│   │
│   ├── lib/                         # Third-party setup
│   │   ├── queryClient.ts           # React Query setup
│   │   ├── axios.ts                 # Axios instance
│   │   └── i18n.ts                  # i18n configuration
│   │
│   ├── pages/                       # Page components
│   │   ├── HomePage/
│   │   │   ├── HomePage.tsx
│   │   │   ├── HomePage.lazy.tsx
│   │   │   └── index.ts
│   │   ├── DashboardPage/
│   │   └── NotFoundPage/
│   │
│   ├── services/                    # Business logic
│   │   ├── api/
│   │   │   ├── client.ts
│   │   │   ├── endpoints.ts
│   │   │   └── types.ts
│   │   └── auth/
│   │       ├── authService.ts
│   │       └── tokenService.ts
│   │
│   ├── store/                       # Global state
│   │   ├── slices/
│   │   │   ├── userSlice.ts
│   │   │   └── uiSlice.ts
│   │   └── index.ts
│   │
│   ├── types/                       # Shared types
│   │   ├── api.types.ts
│   │   ├── user.types.ts
│   │   └── index.ts
│   │
│   ├── utils/                       # Utilities
│   │   ├── formatters/
│   │   │   ├── dateFormatter.ts
│   │   │   └── currencyFormatter.ts
│   │   ├── validators/
│   │   │   └── emailValidator.ts
│   │   ├── constants/
│   │   │   ├── routes.ts
│   │   │   └── config.ts
│   │   └── index.ts
│   │
│   ├── assets/                      # Static assets
│   │   ├── images/
│   │   ├── icons/
│   │   ├── fonts/
│   │   └── styles/
│   │       ├── globals.css
│   │       ├── variables.css
│   │       └── reset.css
│   │
│   ├── test/                        # Test utilities
│   │   ├── setup.ts
│   │   ├── utils.tsx
│   │   └── mocks/
│   │       ├── handlers.ts
│   │       └── data.ts
│   │
│   ├── main.tsx                     # Entry point
│   └── vite-env.d.ts               # Vite types
│
├── .env.development                 # Dev environment variables
├── .env.production                  # Prod environment variables
├── .eslintrc.cjs                    # ESLint config
├── .gitignore
├── .prettierrc                      # Prettier config
├── index.html                       # HTML template
├── package.json
├── tsconfig.json                    # TypeScript config
├── tsconfig.node.json              # TS config for build tools
├── vite.config.ts                  # Vite config
├── vitest.config.ts                # Vitest config
└── README.md
```

## 关键文件内容

### src/main.tsx
```typescript
import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './app/App';
import './assets/styles/globals.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
```

### src/app/App.tsx
```typescript
import { Providers } from './providers';
import { Router } from './router';

export function App() {
  return (
    <Providers>
      <Router />
    </Providers>
  );
}
```

### src/app/providers.tsx
```typescript
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from '@/lib/queryClient';
import { BrowserRouter } from 'react-router-dom';

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        {children}
      </BrowserRouter>
    </QueryClientProvider>
  );
}
```

### src/app/router.tsx
```typescript
import { Routes, Route } from 'react-router-dom';
import { Suspense } from 'react';
import { HomePageLazy } from '@/pages/HomePage';
import { DashboardPageLazy } from '@/pages/DashboardPage';
import { NotFoundPage } from '@/pages/NotFoundPage';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';

export function Router() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <Routes>
        <Route path="/" element={<HomePageLazy />} />
        <Route path="/dashboard" element={<DashboardPageLazy />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
  );
}
```

### src/lib/queryClient.ts
```typescript
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000,   // 10 minutes (formerly cacheTime)
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});
```

### src/lib/axios.ts
```typescript
import axios from 'axios';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

### src/utils/constants/routes.ts
```typescript
export const ROUTES = {
  HOME: '/',
  DASHBOARD: '/dashboard',
  LOGIN: '/login',
  PROFILE: '/profile',
  SETTINGS: '/settings',
} as const;
```

### src/types/api.types.ts
```typescript
export interface ApiResponse<T> {
  data: T;
  message: string;
  success: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  pageSize: number;
  total: number;
}

export interface ApiError {
  message: string;
  code: string;
  details?: unknown;
}
```

### src/测试/setup.ts
```typescript
import { expect, afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import * as matchers from '@testing-library/jest-dom/matchers';

expect.extend(matchers);

afterEach(() => {
  cleanup();
});
```

### src/测试/utils.tsx
```typescript
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

interface AllProvidersProps {
  children: React.ReactNode;
}

function AllProviders({ children }: AllProvidersProps) {
  const testQueryClient = createTestQueryClient();

  return (
    <QueryClientProvider client={testQueryClient}>
      <BrowserRouter>
        {children}
      </BrowserRouter>
    </QueryClientProvider>
  );
}

function customRender(ui: React.ReactElement, options?: RenderOptions) {
  return render(ui, { wrapper: AllProviders, ...options });
}

export * from '@testing-library/react';
export { customRender as render };
```

### vitest.config.ts
```typescript
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    css: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'clover', 'json'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/mockData',
        '**/*.stories.tsx',
      ],
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
});
```

### .eslintrc.cjs
```javascript
module.exports = {
  root: true,
  env: { browser: true, es2020: true },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react-hooks/recommended',
  ],
  ignorePatterns: ['dist', '.eslintrc.cjs'],
  parser: '@typescript-eslint/parser',
  plugins: ['react-refresh'],
  rules: {
    'react-refresh/only-export-components': [
      'warn',
      { allowConstantExport: true },
    ],
    '@typescript-eslint/no-unused-vars': [
      'error',
      { argsIgnorePattern: '^_' },
    ],
  },
};
```

### .prettierrc
```json
{
  "semi": true,
  "singleQuote": true,
  "tabWidth": 2,
  "trailingComma": "es5",
  "printWidth": 100,
  "arrowParens": "always"
}
```

## 创建这个结构

使用提供的脚本生成此结构：

```bash
# Create a component
python scripts/create_component.py Button --type component

# Create a page with lazy loading
python scripts/create_component.py HomePage --type page

# Create a custom hook
python scripts/create_hook.py useAuth --type custom

# Create a feature module
mkdir -p src/features/auth/{components,hooks,api,store}
```

## 应用最佳实践

1. ✅ 基于功能的组织
2. ✅ 相关文件的托管
3. ✅ 用于干净导入的路径别名
4. ✅ 路由延迟加载
5. ✅ 集中配置
6. ✅ 一切都类型安全
7. ✅ 测试实用程序和设置
8. ✅ 一致的命名约定
9. ✅ 关注点分离
10. ✅ 桶出口（index.ts）

## 扩展指南

### 小型应用程序（< 10 个组件）
- 扁平结构在`src/components`
- 单身的`hooks`文件夹
- 最小的组织

### 中型应用程序（10-50 个组件）
- 使用功能文件夹
- 单独的页面
- 共享组件文件夹
- 自定义钩子文件夹

### 大型应用程序（50 多个组件）
- 完整的基于特征的架构
- 领域驱动设计
- 微服务就绪结构
- 考虑 monorepo（nx、turborepo）

## 迁移路径

如果您有现有项目：

1. 为 vite.config.ts 和 tsconfig.json 添加路径别名
2. 创造`features/`文件夹
3. 将相关组件移至功能中
4. 将共享组件提取到`components/`
5. 将钩子提取到`hooks/`
6. 创造`lib/`用于第三方设置
7. 将页面移至`pages/`
8. 逐步更新导入
