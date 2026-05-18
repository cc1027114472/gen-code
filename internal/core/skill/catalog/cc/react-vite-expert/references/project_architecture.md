# React + Vite 项目架构

使用 Vite 构建可扩展、可维护的 React 应用程序的综合指南。

## 最佳文件夹结构

### 基于功能的架构（推荐用于大型应用程序）

```
src/
├── app/                          # App-level configuration
│   ├── App.tsx                   # Root component
│   ├── router.tsx                # Route configuration
│   ├── store.ts                  # Global store setup
│   └── providers.tsx             # Context providers wrapper
│
├── features/                     # Feature modules (business logic)
│   ├── auth/
│   │   ├── components/          # Feature-specific components
│   │   │   ├── LoginForm/
│   │   │   │   ├── LoginForm.tsx
│   │   │   │   ├── LoginForm.types.ts
│   │   │   │   ├── LoginForm.module.css
│   │   │   │   ├── LoginForm.test.tsx
│   │   │   │   └── index.ts
│   │   │   └── RegisterForm/
│   │   ├── hooks/               # Feature-specific hooks
│   │   │   ├── useAuth.ts
│   │   │   └── useLogin.ts
│   │   ├── api/                 # API calls for this feature
│   │   │   ├── authApi.ts
│   │   │   └── authApi.types.ts
│   │   ├── store/               # Feature state (Redux/Zustand)
│   │   │   ├── authSlice.ts
│   │   │   └── authSelectors.ts
│   │   ├── utils/               # Feature-specific utilities
│   │   │   └── validateCredentials.ts
│   │   └── index.ts             # Public API of the feature
│   │
│   ├── dashboard/
│   ├── products/
│   └── settings/
│
├── components/                   # Shared/reusable components
│   ├── ui/                      # Basic UI components
│   │   ├── Button/
│   │   │   ├── Button.tsx
│   │   │   ├── Button.types.ts
│   │   │   ├── Button.module.css
│   │   │   ├── Button.test.tsx
│   │   │   ├── Button.stories.tsx
│   │   │   └── index.ts
│   │   ├── Input/
│   │   ├── Modal/
│   │   └── Card/
│   │
│   ├── layout/                  # Layout components
│   │   ├── Header/
│   │   ├── Footer/
│   │   ├── Sidebar/
│   │   └── PageLayout/
│   │
│   └── form/                    # Form components
│       ├── FormField/
│       ├── FormError/
│       └── FormSubmit/
│
├── hooks/                        # Shared custom hooks
│   ├── useDebounce.ts
│   ├── useLocalStorage.ts
│   ├── useMediaQuery.ts
│   └── usePrevious.ts
│
├── lib/                          # Third-party integrations & setup
│   ├── axios.ts                 # Axios instance with interceptors
│   ├── queryClient.ts           # React Query client
│   ├── i18n.ts                  # i18n configuration
│   └── analytics.ts             # Analytics setup
│
├── pages/                        # Page components (routes)
│   ├── HomePage/
│   │   ├── HomePage.tsx
│   │   ├── HomePage.lazy.tsx    # Lazy-loaded wrapper
│   │   └── index.ts
│   ├── DashboardPage/
│   ├── NotFoundPage/
│   └── index.ts
│
├── services/                     # Business logic & API services
│   ├── api/                     # API clients
│   │   ├── client.ts           # Base API client
│   │   ├── endpoints.ts        # API endpoints
│   │   └── types.ts            # API types
│   ├── auth/                   # Auth service
│   │   ├── authService.ts
│   │   └── tokenService.ts
│   └── storage/                # Storage service
│       └── storageService.ts
│
├── store/                        # Global state management
│   ├── slices/                 # Redux slices or Zustand stores
│   │   ├── userSlice.ts
│   │   └── uiSlice.ts
│   ├── middleware/             # Custom middleware
│   │   └── logger.ts
│   └── index.ts                # Store configuration
│
├── types/                        # Shared TypeScript types
│   ├── api.types.ts            # API response types
│   ├── user.types.ts           # User-related types
│   ├── common.types.ts         # Common types
│   └── index.ts                # Type exports
│
├── utils/                        # Utility functions
│   ├── formatters/             # Data formatters
│   │   ├── dateFormatter.ts
│   │   ├── currencyFormatter.ts
│   │   └── numberFormatter.ts
│   ├── validators/             # Validation functions
│   │   ├── emailValidator.ts
│   │   └── formValidator.ts
│   ├── helpers/                # Helper functions
│   │   ├── arrayHelpers.ts
│   │   └── objectHelpers.ts
│   └── constants/              # Constants
│       ├── routes.ts
│       ├── apiEndpoints.ts
│       └── config.ts
│
├── assets/                       # Static assets
│   ├── images/
│   ├── icons/
│   ├── fonts/
│   └── styles/                 # Global styles
│       ├── globals.css
│       ├── variables.css
│       └── reset.css
│
├── test/                         # Test utilities
│   ├── setup.ts                # Test setup
│   ├── utils.tsx               # Testing utilities
│   ├── mocks/                  # Mock data
│   │   ├── handlers.ts         # MSW handlers
│   │   └── data.ts             # Mock data
│   └── fixtures/               # Test fixtures
│
├── main.tsx                      # Entry point
├── vite-env.d.ts                # Vite types
└── router.tsx                    # Main router (alternative to app/)
```

### 更简单的架构（适用于中小型应用程序）

```
src/
├── components/                   # All components
│   ├── common/                  # Shared components
│   │   ├── Button/
│   │   └── Input/
│   ├── layout/                  # Layout components
│   │   ├── Header/
│   │   └── Footer/
│   └── features/                # Feature components
│       ├── Auth/
│       └── Dashboard/
│
├── hooks/                        # Custom hooks
├── pages/                        # Page components
├── services/                     # API & business logic
├── store/                        # State management
├── types/                        # TypeScript types
├── utils/                        # Utilities
├── assets/                       # Static files
├── main.tsx
└── App.tsx
```

## 命名约定

### 文件和文件夹
```
Component files:       PascalCase     → Button.tsx, UserProfile.tsx
Component folders:     PascalCase     → Button/, UserProfile/
Hook files:           camelCase      → useAuth.ts, useDebounce.ts
Utility files:        camelCase      → formatDate.ts, apiClient.ts
Type files:          camelCase      → user.types.ts, api.types.ts
Style files:         camelCase      → Button.module.css, globals.css
Test files:          match source   → Button.test.tsx, useAuth.test.ts
Story files:         match source   → Button.stories.tsx
```

### 代码
```typescript
// Components: PascalCase
export const Button = () => { }
export const UserProfile = () => { }

// Hooks: camelCase with 'use' prefix
export const useAuth = () => { }
export const useDebounce = () => { }

// Constants: UPPER_SNAKE_CASE
export const API_BASE_URL = 'https://api.example.com';
export const MAX_FILE_SIZE = 5000000;

// Functions: camelCase
export const formatDate = () => { }
export const validateEmail = () => { }

// Types/Interfaces: PascalCase
export interface User { }
export type UserRole = 'admin' | 'user';

// Enums: PascalCase (name) and UPPER_SNAKE_CASE (values)
export enum UserRole {
  ADMIN = 'ADMIN',
  USER = 'USER'
}
```

## 组件组织模式

### 模式一：主机托管（推荐）
每个组件都有自己的文件夹，其中包含所有相关文件：

```
Button/
├── Button.tsx              # Component implementation
├── Button.types.ts         # TypeScript types/interfaces
├── Button.module.css       # Styles (CSS Modules)
├── Button.test.tsx         # Unit tests
├── Button.stories.tsx      # Storybook stories
└── index.ts                # Public API (clean imports)
```

**好处：**
- 方便查找相关文件
- 易于移动/删除功能
- 清晰的界限

### 模式 2：原子设计
按复杂性组织组件：

```
components/
├── atoms/          # Basic building blocks (Button, Input, Label)
├── molecules/      # Simple combinations (FormField, SearchBox)
├── organisms/      # Complex components (Header, ProductCard)
├── templates/      # Page layouts (DashboardTemplate)
└── pages/          # Complete pages (HomePage, DashboardPage)
```

### 模式 3：领域驱动设计
按业务领域整理：

```
src/
├── domains/
│   ├── user/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── services/
│   │   └── types/
│   ├── product/
│   └── order/
```

## 状态管理策略

### 本地状态（useState）
对于不需要共享的特定于组件的状态。

```typescript
// ✅ Good use cases
const [isOpen, setIsOpen] = useState(false);
const [inputValue, setInputValue] = useState('');
const [selectedTab, setSelectedTab] = useState(0);
```

### 提升状态（道具）
用于在兄弟组件之间共享状态。

```typescript
// Parent manages state, children receive via props
function Parent() {
  const [user, setUser] = useState<User | null>(null);

  return (
    <>
      <UserProfile user={user} />
      <UserSettings user={user} onUpdate={setUser} />
    </>
  );
}
```

### 上下文API
对于主题、授权、本地化 - 低频更新。

```typescript
// ✅ Good use cases
const ThemeContext = createContext<Theme>('light');
const AuthContext = createContext<AuthState>(null);
const I18nContext = createContext<I18nState>('en');

// ❌ Avoid for high-frequency updates (causes re-renders)
```

### Zustand（推荐用于大多数应用程序）
轻量级、简单的 API、出色的性能。

```typescript
// store/userStore.ts
import { create } from 'zustand';

interface UserStore {
  user: User | null;
  login: (user: User) => void;
  logout: () => void;
}

export const useUserStore = create<UserStore>((set) => ({
  user: null,
  login: (user) => set({ user }),
  logout: () => set({ user: null }),
}));

// Usage in component
const user = useUserStore((state) => state.user);
const login = useUserStore((state) => state.login);
```

### Redux 工具包
对于具有大量异步逻辑和中间件需求的复杂应用程序。

```typescript
// store/slices/userSlice.ts
import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';

export const fetchUser = createAsyncThunk('user/fetch', async (id: string) => {
  const response = await api.getUser(id);
  return response.data;
});

const userSlice = createSlice({
  name: 'user',
  initialState: { user: null, loading: false },
  reducers: {
    logout: (state) => { state.user = null; },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchUser.pending, (state) => { state.loading = true; })
      .addCase(fetchUser.fulfilled, (state, action) => {
        state.user = action.payload;
        state.loading = false;
      });
  },
});
```

### TanStack 查询（反应查询）
用于服务器状态（API 数据、缓存、同步）。

```typescript
// hooks/useUser.ts
import { useQuery } from '@tanstack/react-query';

export const useUser = (userId: string) => {
  return useQuery({
    queryKey: ['user', userId],
    queryFn: () => api.getUser(userId),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
};

// Usage
const { data: user, isLoading, error } = useUser('123');
```

## 进口策略

### 绝对导入（推荐）
配置路径别名`vite.config.ts`和`tsconfig.json`:

```typescript
// tsconfig.json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "@/components/*": ["src/components/*"],
      "@/hooks/*": ["src/hooks/*"],
      "@/utils/*": ["src/utils/*"],
      "@/types/*": ["src/types/*"]
    }
  }
}

// vite.config.ts
export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@/components': path.resolve(__dirname, './src/components'),
      '@/hooks': path.resolve(__dirname, './src/hooks'),
    },
  },
});

// Usage in files
import { Button } from '@/components/ui/Button';
import { useAuth } from '@/hooks/useAuth';
import { User } from '@/types/user.types';
```

### 桶出口 (index.ts)
为文件夹创建干净的公共 API：

```typescript
// components/ui/index.ts
export { Button } from './Button';
export { Input } from './Input';
export { Modal } from './Modal';

// Usage
import { Button, Input, Modal } from '@/components/ui';

// ⚠️ Warning: Can hurt tree-shaking if not careful
// Only export what's actually public API
```

### 指定导出（推荐）
```typescript
// ✅ Good: Named exports (tree-shakeable)
export const Button = () => { };
export const Input = () => { };

import { Button } from './components';

// ❌ Avoid: Default exports (harder to refactor, not tree-shakeable)
export default Button;
import Button from './components/Button';
```

## 文件大小指南

```
Component file:       < 250 lines (split if larger)
Hook file:           < 100 lines
Utility file:        < 150 lines
Type file:           No limit (just types)
Test file:           < 500 lines

If exceeding limits, consider:
- Breaking into smaller components
- Extracting logic to hooks
- Moving utilities to separate files
- Creating sub-components
```

## 代码组织最佳实践

### 1. 单一职责原则
每个组件/钩子/函数应该做好一件事。

```typescript
// ❌ Bad: Component doing too much
function UserDashboard() {
  // Fetching data
  // Handling forms
  // Managing UI state
  // Rendering complex UI
}

// ✅ Good: Split responsibilities
function UserDashboard() {
  return (
    <DashboardLayout>
      <UserProfile />
      <UserStats />
      <UserActivity />
    </DashboardLayout>
  );
}
```

### 2. 组合优于继承
使用组合来构建复杂的组件。

```typescript
// ✅ Composition pattern
<Card>
  <CardHeader>
    <CardTitle>User Profile</CardTitle>
  </CardHeader>
  <CardBody>
    <UserInfo />
  </CardBody>
</Card>
```

### 3. 容器/展示模式
将逻辑与表示分开。

```typescript
// Presentational (dumb component)
export const UserList = ({ users, onUserClick }) => (
  <ul>
    {users.map(user => (
      <li key={user.id} onClick={() => onUserClick(user)}>
        {user.name}
      </li>
    ))}
  </ul>
);

// Container (smart component)
export const UserListContainer = () => {
  const { data: users } = useUsers();
  const navigate = useNavigate();

  const handleUserClick = (user) => {
    navigate(`/user/${user.id}`);
  };

  return <UserList users={users} onUserClick={handleUserClick} />;
};
```

### 4. 用于逻辑重用的自定义钩子
将可重用逻辑提取到自定义挂钩中。

```typescript
// hooks/useUser.ts
export const useUser = (userId: string) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchUser(userId).then(setUser).finally(() => setLoading(false));
  }, [userId]);

  return { user, loading };
};

// Usage in multiple components
const { user, loading } = useUser('123');
```

## 决策矩阵

### 何时创建新功能模块？
- ✅ 有 3 个以上组件
- ✅ 有自己的状态管理
- ✅ 拥有专用的 API 端点
- ✅ 代表独特的业务能力

### 何时使用上下文与道具？
- **Props**：默认选择，显式，类型安全
- **上下文**：避免道具钻探（4+级别）、主题、auth、i18n

### 何时使用 Redux 与 Zustand？
- **Zustand**：大多数应用程序，更简单的 API，更少的样板文件
- **Redux**：复杂的应用程序，需要中间件、开发工具、时间旅行调试

### 何时拆分组件？
- 🚩 文件 > 250 行
- 🚩 多重责任
- 🚩 可重复使用的部件
- 🚩 难以测试
- 🚩 表现不佳（需要备注）
