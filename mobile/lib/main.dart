import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

void main() => runApp(const AnNgonApp());

const apiBaseUrl = 'http://127.0.0.1:8080/api/v1';

final authController = AuthController(ApiClient(baseUrl: apiBaseUrl));

void openCreateReview(BuildContext context) {
  Navigator.of(
    context,
  ).push(MaterialPageRoute(builder: (_) => const CreateReviewScreenV2()));
}

class ApiException implements Exception {
  const ApiException(this.message);

  final String message;

  @override
  String toString() => message;
}

class ApiClient {
  ApiClient({required this.baseUrl, http.Client? client})
    : _client = client ?? http.Client();

  final String baseUrl;
  final http.Client _client;
  String? token;

  Future<Map<String, dynamic>> post(
    String path, {
    Map<String, dynamic>? body,
    bool auth = false,
  }) async {
    final res = await _client.post(
      Uri.parse('$baseUrl$path'),
      headers: _headers(auth),
      body: jsonEncode(body ?? const {}),
    );
    return _decode(res);
  }

  Future<Map<String, dynamic>> get(String path, {bool auth = false}) async {
    final res = await _client.get(
      Uri.parse('$baseUrl$path'),
      headers: _headers(auth),
    );
    return _decode(res);
  }

  Future<List<dynamic>> getList(String path, {bool auth = false}) async {
    final res = await _client.get(
      Uri.parse('$baseUrl$path'),
      headers: _headers(auth),
    );
    return _decodeList(res);
  }

  Map<String, String> _headers(bool auth) {
    return {
      'Content-Type': 'application/json; charset=utf-8',
      if (auth && token != null) 'Authorization': 'Bearer $token',
    };
  }

  Map<String, dynamic> _decode(http.Response res) {
    final data = _decodeEnvelope(res);
    return data as Map<String, dynamic>? ?? const {};
  }

  List<dynamic> _decodeList(http.Response res) {
    final data = _decodeEnvelope(res);
    return data as List<dynamic>? ?? const [];
  }

  dynamic _decodeEnvelope(http.Response res) {
    if (res.statusCode == 204) return const {};
    final decoded =
        jsonDecode(utf8.decode(res.bodyBytes)) as Map<String, dynamic>;
    final error = decoded['error'];
    if (res.statusCode >= 400 || error != null) {
      final message = error is Map<String, dynamic>
          ? error['message']?.toString() ?? 'Có lỗi xảy ra'
          : 'Có lỗi xảy ra';
      throw ApiException(message);
    }
    return decoded['data'];
  }

  Future<AuthSession> login(String login, String password) async {
    final data = await post(
      '/auth/login',
      body: {'login': login, 'password': password},
    );
    final session = AuthSession.fromJson(data);
    token = session.token;
    return session;
  }

  Future<AuthSession> register({
    required String username,
    required String email,
    required String password,
    required String displayName,
  }) async {
    final data = await post(
      '/auth/register',
      body: {
        'username': username,
        'email': email,
        'password': password,
        'display_name': displayName,
      },
    );
    final session = AuthSession.fromJson(data);
    token = session.token;
    return session;
  }

  Future<AuthUser> me() async {
    final data = await get('/me', auth: true);
    return AuthUser.fromJson(data);
  }

  Future<void> logout() async {
    if (token != null) {
      await post('/auth/logout', auth: true);
    }
    token = null;
  }

  Future<List<FoodPost>> listPosts() async {
    final rows = await getList('/posts?limit=20');
    return rows
        .whereType<Map<String, dynamic>>()
        .map(FoodPost.fromApi)
        .toList();
  }

  Future<int?> createPlaceByName(String name) async {
    final clean = name.trim();
    if (clean.isEmpty) return null;
    final data = await post('/places', body: {'name': clean}, auth: true);
    return (data['id'] as num?)?.toInt();
  }

  Future<FoodPost> createPost({
    required String content,
    required List<String> hashtags,
    int? placeId,
  }) async {
    final data = await post(
      '/posts',
      auth: true,
      body: {
        'content': content,
        'is_sponsored': false,
        'media_ids': <int>[],
        'place_id': placeId,
        'hashtags': hashtags,
      },
    );
    return FoodPost.fromApi(data);
  }
}

class AuthSession {
  const AuthSession({required this.token, required this.user});

  final String token;
  final AuthUser user;

  factory AuthSession.fromJson(Map<String, dynamic> json) {
    return AuthSession(
      token: json['token']?.toString() ?? '',
      user: AuthUser.fromJson(
        json['user'] as Map<String, dynamic>? ?? const {},
      ),
    );
  }
}

class AuthUser {
  const AuthUser({
    required this.id,
    required this.username,
    required this.email,
    required this.displayName,
  });

  final int id;
  final String username;
  final String email;
  final String displayName;

  factory AuthUser.fromJson(Map<String, dynamic> json) {
    return AuthUser(
      id: (json['id'] as num?)?.toInt() ?? 0,
      username: json['username']?.toString() ?? '',
      email: json['email']?.toString() ?? '',
      displayName: json['display_name']?.toString() ?? '',
    );
  }
}

class AuthController extends ChangeNotifier {
  AuthController(this.api);

  final ApiClient api;
  AuthSession? session;
  bool loading = false;
  String? error;

  bool get isSignedIn => session != null;
  AuthUser? get user => session?.user;

  Future<void> login(String login, String password) async {
    await _run(() async {
      session = await api.login(login.trim(), password);
    });
  }

  Future<void> register({
    required String username,
    required String email,
    required String password,
    required String displayName,
  }) async {
    await _run(() async {
      session = await api.register(
        username: username.trim(),
        email: email.trim(),
        password: password,
        displayName: displayName.trim(),
      );
    });
  }

  Future<void> logout() async {
    await _run(() async {
      await api.logout();
      session = null;
    });
  }

  Future<void> _run(Future<void> Function() action) async {
    loading = true;
    error = null;
    notifyListeners();
    try {
      await action();
    } on ApiException catch (e) {
      error = e.message;
    } catch (_) {
      error = 'Không kết nối được backend. Kiểm tra API đang chạy ở :8080.';
    } finally {
      loading = false;
      notifyListeners();
    }
  }
}

class AnNgonApp extends StatelessWidget {
  const AnNgonApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'AnNgon',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light(),
      home: const MobilePreviewFrame(child: AuthGate()),
    );
  }
}

class AuthGate extends StatelessWidget {
  const AuthGate({super.key});

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: authController,
      builder: (context, _) {
        if (authController.isSignedIn) {
          return const HomeShell();
        }
        return const LoginScreen();
      },
    );
  }
}

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final loginController = TextEditingController();
  final usernameController = TextEditingController();
  final emailController = TextEditingController();
  final displayNameController = TextEditingController();
  final passwordController = TextEditingController();
  bool registerMode = false;

  @override
  void dispose() {
    loginController.dispose();
    usernameController.dispose();
    emailController.dispose();
    displayNameController.dispose();
    passwordController.dispose();
    super.dispose();
  }

  Future<void> submit() async {
    if (registerMode) {
      await authController.register(
        username: usernameController.text,
        email: emailController.text,
        displayName: displayNameController.text,
        password: passwordController.text,
      );
    } else {
      await authController.login(loginController.text, passwordController.text);
    }

    if (!mounted) return;
    final error = authController.error;
    if (error != null) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(error)));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.paper,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(18, 28, 18, 24),
          children: [
            const AuthHeroMark(),
            const SizedBox(height: 26),
            Text(
              registerMode ? 'Tạo tài khoản review' : 'Chào mừng quay lại',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 6),
            Text(
              registerMode
                  ? 'Tham gia cộng đồng review món ngon địa phương.'
                  : 'Đăng nhập để lưu bài, đăng review và theo dõi địa điểm.',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 22),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(14),
                child: Column(
                  children: [
                    AuthModeSwitch(
                      registerMode: registerMode,
                      onChanged: (value) =>
                          setState(() => registerMode = value),
                    ),
                    const SizedBox(height: 14),
                    if (registerMode) ...[
                      AppTextField(
                        controller: usernameController,
                        icon: Icons.alternate_email_rounded,
                        label: 'Username',
                        hint: 'duy_foodie',
                      ),
                      const SizedBox(height: 10),
                      AppTextField(
                        controller: displayNameController,
                        icon: Icons.badge_outlined,
                        label: 'Tên hiển thị',
                        hint: 'Duy ăn Hải Phòng',
                      ),
                      const SizedBox(height: 10),
                      AppTextField(
                        controller: emailController,
                        icon: Icons.mail_outline_rounded,
                        label: 'Email',
                        hint: 'you@example.com',
                      ),
                    ] else ...[
                      AppTextField(
                        controller: loginController,
                        icon: Icons.person_outline_rounded,
                        label: 'Email hoặc username',
                        hint: 'you@example.com',
                      ),
                    ],
                    const SizedBox(height: 10),
                    AppTextField(
                      controller: passwordController,
                      icon: Icons.lock_outline_rounded,
                      label: 'Mật khẩu',
                      hint: 'Ít nhất 8 ký tự',
                      obscure: true,
                    ),
                    const SizedBox(height: 16),
                    AnimatedBuilder(
                      animation: authController,
                      builder: (context, _) {
                        return FilledButton(
                          onPressed: authController.loading ? null : submit,
                          style: FilledButton.styleFrom(
                            minimumSize: const Size.fromHeight(46),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(17),
                            ),
                          ),
                          child: authController.loading
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                    color: Colors.white,
                                  ),
                                )
                              : Text(
                                  registerMode ? 'Tạo tài khoản' : 'Đăng nhập',
                                ),
                        );
                      },
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 14),
            Text(
              'Bản đầu dùng session token từ backend. Secure storage, OTP/email/SMS thật sẽ nối ở bước sau.',
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(fontSize: 12),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}

class AuthHeroMark extends StatelessWidget {
  const AuthHeroMark({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 150,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppTheme.brand,
        borderRadius: BorderRadius.circular(30),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: .16),
              borderRadius: BorderRadius.circular(16),
            ),
            child: const Icon(Icons.rate_review_rounded, color: Colors.white),
          ),
          const Spacer(),
          const Text(
            'Review món ngon,\nđúng chất địa phương.',
            style: TextStyle(
              color: Colors.white,
              fontSize: 20,
              fontWeight: FontWeight.w900,
              height: 1.15,
            ),
          ),
        ],
      ),
    );
  }
}

class AuthModeSwitch extends StatelessWidget {
  const AuthModeSwitch({
    super.key,
    required this.registerMode,
    required this.onChanged,
  });

  final bool registerMode;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF3EB),
        borderRadius: BorderRadius.circular(18),
      ),
      child: Row(
        children: [
          Expanded(
            child: AuthModeTab(
              label: 'Đăng nhập',
              selected: !registerMode,
              onTap: () => onChanged(false),
            ),
          ),
          Expanded(
            child: AuthModeTab(
              label: 'Đăng ký',
              selected: registerMode,
              onTap: () => onChanged(true),
            ),
          ),
        ],
      ),
    );
  }
}

class AuthModeTab extends StatelessWidget {
  const AuthModeTab({
    super.key,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(14),
      child: Container(
        height: 34,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: selected ? Colors.white : Colors.transparent,
          borderRadius: BorderRadius.circular(14),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: selected ? AppTheme.brand : AppTheme.muted,
            fontWeight: FontWeight.w900,
            fontSize: 12.5,
          ),
        ),
      ),
    );
  }
}

class AppTextField extends StatelessWidget {
  const AppTextField({
    super.key,
    required this.controller,
    required this.icon,
    required this.label,
    required this.hint,
    this.obscure = false,
  });

  final TextEditingController controller;
  final IconData icon;
  final String label;
  final String hint;
  final bool obscure;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      obscureText: obscure,
      decoration: InputDecoration(
        labelText: label,
        hintText: hint,
        prefixIcon: Icon(icon, size: 19),
        filled: true,
        fillColor: const Color(0xFFFFFBF7),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(17),
          borderSide: const BorderSide(color: AppTheme.line),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(17),
          borderSide: const BorderSide(color: AppTheme.line),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(17),
          borderSide: const BorderSide(color: AppTheme.brand),
        ),
      ),
    );
  }
}

class AppTheme {
  static const brand = Color(0xFFF45A35);
  static const brandDark = Color(0xFFB73B25);
  static const brandSoft = Color(0xFFFFE8DE);
  static const trust = Color(0xFF25A86A);
  static const accent = Color(0xFFFFC96B);
  static const ink = Color(0xFF2A211B);
  static const muted = Color(0xFF8A776C);
  static const paper = Color(0xFFFFF8F0);
  static const surface = Color(0xFFFFFFFF);
  static const line = Color(0xFFF0E4D8);

  static ThemeData light() {
    final scheme = ColorScheme.fromSeed(
      seedColor: brand,
      brightness: Brightness.light,
      primary: brand,
      secondary: accent,
      surface: surface,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: paper,
      fontFamily: 'Roboto',
      textTheme: const TextTheme(
        headlineSmall: TextStyle(
          color: ink,
          fontSize: 25,
          fontWeight: FontWeight.w900,
          height: 1.12,
        ),
        titleLarge: TextStyle(
          color: ink,
          fontSize: 20,
          fontWeight: FontWeight.w900,
        ),
        titleMedium: TextStyle(
          color: ink,
          fontSize: 16,
          fontWeight: FontWeight.w800,
        ),
        bodyLarge: TextStyle(color: ink, fontSize: 15, height: 1.38),
        bodyMedium: TextStyle(color: muted, fontSize: 13.5, height: 1.35),
        labelLarge: TextStyle(fontSize: 14, fontWeight: FontWeight.w800),
      ),
      appBarTheme: const AppBarTheme(
        elevation: 0,
        centerTitle: false,
        backgroundColor: paper,
        foregroundColor: ink,
        titleTextStyle: TextStyle(
          color: ink,
          fontSize: 24,
          fontWeight: FontWeight.w900,
          letterSpacing: 0,
        ),
      ),
      cardTheme: CardThemeData(
        color: surface,
        elevation: 1,
        shadowColor: const Color(0x16B56B45),
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(26),
          side: const BorderSide(color: line),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: brand,
          foregroundColor: Colors.white,
          minimumSize: const Size(48, 44),
          padding: const EdgeInsets.symmetric(horizontal: 18),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
          ),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        height: 58,
        elevation: 0,
        backgroundColor: surface,
        indicatorColor: Colors.transparent,
        labelTextStyle: WidgetStateProperty.resolveWith(
          (states) => TextStyle(
            color: states.contains(WidgetState.selected) ? brand : muted,
            fontSize: 11.5,
            fontWeight: states.contains(WidgetState.selected)
                ? FontWeight.w900
                : FontWeight.w600,
          ),
        ),
      ),
    );
  }
}

class HomeShell extends StatefulWidget {
  const HomeShell({super.key});

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  int index = 0;

  final pages = const [
    FeedScreen(),
    DiscoverScreen(),
    SavedScreen(),
    NotificationScreen(),
    ProfileScreen(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: pages[index],
      floatingActionButton: index == 0
          ? Container(
              width: 42,
              height: 42,
              decoration: BoxDecoration(
                color: AppTheme.brand,
                borderRadius: BorderRadius.circular(18),
                boxShadow: const [
                  BoxShadow(
                    color: Color(0x26F45A35),
                    blurRadius: 18,
                    offset: Offset(0, 8),
                  ),
                ],
              ),
              child: IconButton(
                onPressed: () => openCreateReview(context),
                tooltip: 'Đăng review',
                icon: const Icon(
                  Icons.add_rounded,
                  color: Colors.white,
                  size: 24,
                ),
                padding: EdgeInsets.zero,
                style: IconButton.styleFrom(
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  minimumSize: const Size(42, 42),
                ),
              ),
            )
          : null,
      bottomNavigationBar: NavigationBar(
        labelBehavior: NavigationDestinationLabelBehavior.onlyShowSelected,
        selectedIndex: index,
        onDestinationSelected: (value) => setState(() => index = value),
        destinations: const [
          NavigationDestination(
            icon: PremiumNavIcon(icon: Icons.home_outlined),
            selectedIcon: PremiumNavIcon(icon: Icons.home_rounded, selected: true),
            label: 'Feed',
          ),
          NavigationDestination(
            icon: PremiumNavIcon(icon: Icons.explore_outlined),
            selectedIcon: PremiumNavIcon(icon: Icons.explore_rounded, selected: true),
            label: 'Khám phá',
          ),
          NavigationDestination(
            icon: PremiumNavIcon(icon: Icons.bookmark_border_rounded),
            selectedIcon: PremiumNavIcon(icon: Icons.bookmark_rounded, selected: true),
            label: 'Đã lưu',
          ),
          NavigationDestination(
            icon: PremiumNavIcon(icon: Icons.notifications_none_rounded),
            selectedIcon: PremiumNavIcon(icon: Icons.notifications_rounded, selected: true),
            label: 'Tin báo',
          ),
          NavigationDestination(
            icon: PremiumNavIcon(icon: Icons.person_outline_rounded),
            selectedIcon: PremiumNavIcon(icon: Icons.person_rounded, selected: true),
            label: 'Tôi',
          ),
        ],
      ),
    );
  }
}

class MobilePreviewFrame extends StatelessWidget {
  const MobilePreviewFrame({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth <= 520) {
          return child;
        }

        return ColoredBox(
          color: const Color(0xFFF3ECE4),
          child: Center(
            child: Container(
              width: 430,
              height: double.infinity,
              decoration: const BoxDecoration(
                color: AppTheme.paper,
                boxShadow: [
                  BoxShadow(
                    color: Color(0x22000000),
                    blurRadius: 36,
                    offset: Offset(0, 18),
                  ),
                ],
              ),
              child: child,
            ),
          ),
        );
      },
    );
  }
}

class PremiumNavIcon extends StatelessWidget {
  const PremiumNavIcon({super.key, required this.icon, this.selected = false});

  final IconData icon;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    return AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      curve: Curves.easeOutCubic,
      width: selected ? 34 : 30,
      height: 30,
      decoration: BoxDecoration(
        color: selected ? AppTheme.brandSoft : Colors.transparent,
        borderRadius: BorderRadius.circular(14),
      ),
      child: Icon(
        icon,
        size: selected ? 20 : 19,
        color: selected ? AppTheme.brand : AppTheme.muted,
      ),
    );
  }
}

class FeedScreen extends StatefulWidget {
  const FeedScreen({super.key});

  @override
  State<FeedScreen> createState() => _FeedScreenState();
}

class _FeedScreenState extends State<FeedScreen> {
  late Future<List<FoodPost>> postsFuture;

  @override
  void initState() {
    super.initState();
    postsFuture = authController.api.listPosts();
  }

  void reload() {
    setState(() {
      postsFuture = authController.api.listPosts();
    });
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      bottom: false,
      child: CustomScrollView(
        slivers: [
          const SliverAppBar(
            pinned: true,
            floating: true,
            title: AppHeaderMark(),
            actions: [
              AppIconButton(icon: Icons.search_rounded, tooltip: 'Tìm kiếm'),
              AppIconButton(icon: Icons.tune_rounded, tooltip: 'Bộ lọc'),
              SizedBox(width: 8),
            ],
          ),
          const SliverToBoxAdapter(child: LocationHero()),
          const SliverToBoxAdapter(child: CuisineShortcuts()),
          const SliverToBoxAdapter(child: ReviewComposer()),
          const SliverToBoxAdapter(child: FeedTabs()),
          SliverToBoxAdapter(
            child: FutureBuilder<List<FoodPost>>(
              future: postsFuture,
              builder: (context, snapshot) {
                final loading = snapshot.connectionState != ConnectionState.done;
                final hasError = snapshot.hasError;
                final posts = snapshot.data?.isNotEmpty == true
                    ? snapshot.data!
                    : samplePosts;

                return Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 98),
                  child: Column(
                    children: [
                      if (loading)
                        const FeedStateCard(
                          icon: Icons.sync_rounded,
                          title: 'Đang tải review mới...',
                          subtitle: 'Mình đang gọi API /posts từ backend.',
                        )
                      else if (hasError)
                        FeedStateCard(
                          icon: Icons.wifi_off_rounded,
                          title: 'Chưa lấy được feed thật',
                          subtitle:
                              'Backend chưa chạy hoặc chưa đăng nhập đúng API. Đang hiển thị mẫu để bạn vẫn xem UI.',
                          actionLabel: 'Thử lại',
                          onAction: reload,
                        )
                      else if (snapshot.data?.isEmpty == true)
                        const FeedStateCard(
                          icon: Icons.ramen_dining_rounded,
                          title: 'Feed thật đang trống',
                          subtitle:
                              'Đăng review đầu tiên để thay phần dữ liệu mẫu này.',
                        ),
                      if (loading || hasError || snapshot.data?.isEmpty == true)
                        const SizedBox(height: 14),
                      for (var i = 0; i < posts.length; i++) ...[
                        FoodPostCard(post: posts[i]),
                        if (i != posts.length - 1) const SizedBox(height: 16),
                      ],
                    ],
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class FeedStateCard extends StatelessWidget {
  const FeedStateCard({
    super.key,
    required this.icon,
    required this.title,
    required this.subtitle,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: AppTheme.brandSoft,
                borderRadius: BorderRadius.circular(15),
              ),
              child: Icon(icon, color: AppTheme.brand, size: 20),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 3),
                  Text(subtitle, style: Theme.of(context).textTheme.bodyMedium),
                ],
              ),
            ),
            if (actionLabel != null)
              TextButton(onPressed: onAction, child: Text(actionLabel!)),
          ],
        ),
      ),
    );
  }
}

class AppHeaderMark extends StatelessWidget {
  const AppHeaderMark({super.key});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 42,
          height: 34,
          padding: const EdgeInsets.all(5),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: AppTheme.line),
          ),
          child: Row(
            children: [
              Container(
                width: 12,
                height: 24,
                decoration: BoxDecoration(
                  color: AppTheme.brand,
                  borderRadius: BorderRadius.circular(9),
                ),
              ),
              const SizedBox(width: 4),
              Expanded(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Container(
                      height: 8,
                      decoration: BoxDecoration(
                        color: AppTheme.accent,
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    const SizedBox(height: 4),
                    Container(
                      height: 8,
                      decoration: BoxDecoration(
                        color: AppTheme.brandSoft,
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class AppIconButton extends StatelessWidget {
  const AppIconButton({super.key, required this.icon, required this.tooltip});

  final IconData icon;
  final String tooltip;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(left: 4),
      child: Tooltip(
        message: tooltip,
        child: InkWell(
          onTap: () {},
          borderRadius: BorderRadius.circular(13),
          child: Ink(
            width: 34,
            height: 34,
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: .72),
              borderRadius: BorderRadius.circular(13),
              border: Border.all(color: AppTheme.line.withValues(alpha: .72)),
            ),
            child: Icon(icon, size: 18, color: AppTheme.ink),
          ),
        ),
      ),
    );
  }
}

class PremiumRoundIcon extends StatelessWidget {
  const PremiumRoundIcon({
    super.key,
    required this.icon,
    this.size = 34,
    this.iconSize = 18,
    this.foreground = AppTheme.brand,
    this.background = AppTheme.brandSoft,
  });

  final IconData icon;
  final double size;
  final double iconSize;
  final Color foreground;
  final Color background;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(size * .38),
      ),
      child: Icon(icon, color: foreground, size: iconSize),
    );
  }
}

class LocationHero extends StatelessWidget {
  const LocationHero({super.key});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
      child: Container(
        padding: const EdgeInsets.fromLTRB(14, 14, 14, 15),
        decoration: BoxDecoration(
          color: AppTheme.brand,
          borderRadius: BorderRadius.circular(26),
          boxShadow: const [
            BoxShadow(
              color: Color(0x22F45A35),
              blurRadius: 28,
              offset: Offset(0, 14),
            ),
          ],
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: _LocationSelector(
                    city: 'Hải Phòng',
                    subtitle: 'Đang xem review tại',
                  ),
                ),
                const SizedBox(width: 8),
                Container(
                  width: 38,
                  height: 38,
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.13),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: const Icon(
                    Icons.favorite_border_rounded,
                    color: Colors.white,
                    size: 21,
                  ),
                ),
                const SizedBox(width: 6),
                Container(
                  width: 38,
                  height: 38,
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.13),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: const Icon(
                    Icons.receipt_long_rounded,
                    color: Colors.white,
                    size: 21,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            const _SearchBarPreview(),
            const SizedBox(height: 12),
            const Row(
              children: [
                Expanded(
                  child: ServiceModePill(
                    selected: true,
                    icon: Icons.rate_review_rounded,
                    label: 'Review món',
                  ),
                ),
                SizedBox(width: 10),
                Expanded(
                  child: ServiceModePill(
                    selected: false,
                    icon: Icons.restaurant_rounded,
                    label: 'Quán ngon',
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _LocationSelector extends StatelessWidget {
  const _LocationSelector({required this.city, required this.subtitle});

  final String city;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(15),
        border: Border.all(color: Colors.white.withValues(alpha: 0.16)),
      ),
      child: Row(
        children: [
          const Icon(Icons.location_on_rounded, color: Colors.white, size: 18),
          const SizedBox(width: 7),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  city,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 14,
                    fontWeight: FontWeight.w900,
                  ),
                ),
                Text(
                  subtitle,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: Colors.white.withValues(alpha: 0.72),
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ),
          const Icon(
            Icons.keyboard_arrow_down_rounded,
            color: Colors.white,
            size: 20,
          ),
        ],
      ),
    );
  }
}

class ServiceModePill extends StatelessWidget {
  const ServiceModePill({
    super.key,
    required this.selected,
    required this.icon,
    required this.label,
  });

  final bool selected;
  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 38,
      decoration: BoxDecoration(
        color: selected ? Colors.white : Colors.white.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(15),
        border: Border.all(color: Colors.white.withValues(alpha: 0.18)),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 16, color: selected ? AppTheme.brand : Colors.white),
          const SizedBox(width: 6),
          Text(
            label,
            style: TextStyle(
              color: selected ? AppTheme.brand : Colors.white,
              fontSize: 12.5,
              fontWeight: FontWeight.w900,
            ),
          ),
        ],
      ),
    );
  }
}

class _SearchBarPreview extends StatelessWidget {
  const _SearchBarPreview();

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 42,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          const Icon(Icons.search_rounded, color: AppTheme.brand, size: 20),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              'Tìm bánh đa cua, bún cá, quán gần bạn...',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }
}

class CuisineShortcuts extends StatelessWidget {
  const CuisineShortcuts({super.key});

  @override
  Widget build(BuildContext context) {
    final items = [
      ('Món hot', Icons.local_fire_department_rounded),
      ('Gần tôi', Icons.near_me_rounded),
      ('Ăn sáng', Icons.wb_twilight_rounded),
      ('Hải sản', Icons.set_meal_rounded),
      ('Quán mới', Icons.fiber_new_rounded),
    ];

    return SizedBox(
      height: 40,
      child: ListView.separated(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        scrollDirection: Axis.horizontal,
        itemCount: items.length,
        separatorBuilder: (_, _) => const SizedBox(width: 7),
        itemBuilder: (context, index) {
          final selected = index == 0;
          return ChoiceChip(
            selected: selected,
            showCheckmark: false,
            label: Text(items[index].$1),
            avatar: Icon(items[index].$2, size: 15),
            selectedColor: AppTheme.brandSoft,
            backgroundColor: Colors.white,
            side: BorderSide(
              color: selected ? const Color(0xFFFFC2AD) : AppTheme.line,
            ),
            labelStyle: TextStyle(
              color: selected ? AppTheme.brand : AppTheme.muted,
              fontWeight: FontWeight.w900,
              fontSize: 12.2,
            ),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(14),
            ),
            onSelected: (_) {},
          );
        },
      ),
    );
  }
}

class ReviewComposer extends StatelessWidget {
  const ReviewComposer({super.key});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 6, 16, 4),
      child: InkWell(
        onTap: () => openCreateReview(context),
        borderRadius: BorderRadius.circular(24),
        child: Ink(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(24),
            border: Border.all(color: AppTheme.line),
          ),
          child: Row(
            children: [
              const CircleAvatar(
                radius: 19,
                backgroundColor: AppTheme.brandSoft,
                child: Icon(Icons.person_rounded, color: AppTheme.brand),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Bạn vừa ăn gì ngon?',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Đăng review, thêm ảnh, gắn quán ở Hải Phòng',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  ],
                ),
              ),
              Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(
                  color: AppTheme.brand,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(
                  Icons.add_rounded,
                  color: Colors.white,
                  size: 22,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class FeedTabs extends StatelessWidget {
  const FeedTabs({super.key});

  @override
  Widget build(BuildContext context) {
    final tabs = ['Mới nhất', 'Đang theo dõi', 'Gần bạn'];

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 10, 16, 0),
      child: Row(
        children: [
          for (var i = 0; i < tabs.length; i++) ...[
            Expanded(
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 180),
                height: 34,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: i == 0 ? AppTheme.brand : Colors.white,
                  borderRadius: BorderRadius.circular(18),
                  border: Border.all(
                    color: i == 0 ? AppTheme.brand : AppTheme.line,
                  ),
                ),
                child: Text(
                  tabs[i],
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: i == 0 ? Colors.white : AppTheme.ink,
                    fontSize: 12,
                    fontWeight: FontWeight.w900,
                  ),
                ),
              ),
            ),
            if (i < tabs.length - 1) const SizedBox(width: 8),
          ],
        ],
      ),
    );
  }
}

class TrendingAreas extends StatelessWidget {
  const TrendingAreas({super.key});

  @override
  Widget build(BuildContext context) {
    final areas = [
      ('Lạch Tray', '128 review mới'),
      ('Cát Bi', 'Bún cá đang hot'),
      ('Đồ Sơn', 'Hải sản cuối tuần'),
    ];

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 14, 0, 0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(right: 16),
            child: Row(
              children: [
                Text(
                  'Khu vực nổi bật',
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const Spacer(),
                Text(
                  'Xem tất cả',
                  style: TextStyle(
                    color: AppTheme.brand,
                    fontWeight: FontWeight.w900,
                    fontSize: 12.5,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 10),
          SizedBox(
            height: 74,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: areas.length,
              separatorBuilder: (_, _) => const SizedBox(width: 10),
              itemBuilder: (context, index) =>
                  AreaCard(title: areas[index].$1, subtitle: areas[index].$2),
            ),
          ),
        ],
      ),
    );
  }
}

class AreaCard extends StatelessWidget {
  const AreaCard({super.key, required this.title, required this.subtitle});

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 168,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: AppTheme.line),
      ),
      child: Row(
        children: [
          const PremiumRoundIcon(icon: Icons.place_outlined, size: 36),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: AppTheme.ink,
                    fontWeight: FontWeight.w900,
                  ),
                ),
                const SizedBox(height: 3),
                Text(
                  subtitle,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class FoodPostCard extends StatelessWidget {
  const FoodPostCard({super.key, required this.post});

  final FoodPost post;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(
                  radius: 22,
                  backgroundColor: post.avatarColor,
                  child: Text(
                    post.author.characters.first,
                    style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.w900,
                    ),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        post.author,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 2),
                      Text(
                        '${post.city} • ${post.area} • ${post.timeAgo}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ],
                  ),
                ),
                const MoreDotButton(),
              ],
            ),
            const SizedBox(height: 12),
            Text(post.content, style: Theme.of(context).textTheme.bodyLarge),
            const SizedBox(height: 12),
            PhotoGrid(post: post),
            const SizedBox(height: 10),
            TrustSummary(post: post),
            const SizedBox(height: 10),
            Wrap(
              spacing: 7,
              runSpacing: 7,
              children: [
                SoftPill(
                  icon: Icons.payments_rounded,
                  label: post.price,
                  tone: PillTone.price,
                ),
                SoftPill(
                  icon: Icons.near_me_rounded,
                  label: post.distance,
                  tone: PillTone.distance,
                ),
                ...post.tags.map(
                  (tag) => SoftPill(label: tag, tone: PillTone.tag),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                ActionChipButton(
                  icon: Icons.favorite_border_rounded,
                  label: post.likes,
                ),
                const SizedBox(width: 8),
                ActionChipButton(
                  icon: Icons.mode_comment_outlined,
                  label: post.comments,
                ),
                const SizedBox(width: 8),
                const ActionChipButton(
                  icon: Icons.bookmark_border_rounded,
                  label: 'Lưu',
                ),
                const Spacer(),
                const MoreDotButton(),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class TrustSummary extends StatelessWidget {
  const TrustSummary({super.key, required this.post});

  final FoodPost post;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
          decoration: BoxDecoration(
            color: const Color(0xFFE9F7EF),
            borderRadius: BorderRadius.circular(11),
          ),
          child: Row(
            children: [
              const Icon(
                Icons.verified_user_outlined,
                size: 14,
                color: AppTheme.trust,
              ),
              const SizedBox(width: 4),
              Text(
                'Tin cậy ${post.trustScore}',
                style: const TextStyle(
                  color: AppTheme.trust,
                  fontSize: 11.5,
                  fontWeight: FontWeight.w900,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 10),
        Text(
          '${post.reviewCount} lượt đánh giá',
          style: Theme.of(
            context,
          ).textTheme.bodyMedium?.copyWith(fontSize: 11.5),
        ),
      ],
    );
  }
}

class PhotoGrid extends StatelessWidget {
  const PhotoGrid({super.key, required this.post});

  final FoodPost post;

  @override
  Widget build(BuildContext context) {
    final count = post.photoColors.length.clamp(1, 6);
    final colors = post.photoColors.take(count).toList();

    if (count == 1) {
      return AspectRatio(
        aspectRatio: 1.45,
        child: FoodPhotoTile(
          color: colors[0],
          label: post.dishName,
          placeLabel: post.place,
        ),
      );
    }

    if (count == 2) {
      return AspectRatio(
        aspectRatio: 1.85,
        child: Row(
          children: [
            Expanded(
              child: FoodPhotoTile(color: colors[0], placeLabel: post.place),
            ),
            const SizedBox(width: 6),
            Expanded(child: FoodPhotoTile(color: colors[1])),
          ],
        ),
      );
    }

    if (count == 3) {
      return AspectRatio(
        aspectRatio: 1.68,
        child: Row(
          children: [
            Expanded(
              flex: 7,
              child: FoodPhotoTile(
                color: colors[0],
                label: post.dishName,
                placeLabel: post.place,
              ),
            ),
            const SizedBox(width: 6),
            Expanded(
              flex: 5,
              child: Column(
                children: [
                  Expanded(child: FoodPhotoTile(color: colors[1])),
                  const SizedBox(height: 6),
                  Expanded(child: FoodPhotoTile(color: colors[2])),
                ],
              ),
            ),
          ],
        ),
      );
    }

    if (count == 4) {
      return AspectRatio(aspectRatio: 1.32, child: _GridRows(colors: colors));
    }

    return AspectRatio(
      aspectRatio: 1.22,
      child: Column(
        children: [
          Expanded(
            child: Row(
              children: [
                Expanded(
                  child: FoodPhotoTile(
                    color: colors[0],
                    placeLabel: post.place,
                  ),
                ),
                const SizedBox(width: 6),
                Expanded(child: FoodPhotoTile(color: colors[1])),
              ],
            ),
          ),
          const SizedBox(height: 6),
          Expanded(
            child: Row(
              children: [
                for (var i = 2; i < count; i++) ...[
                  Expanded(
                    child: FoodPhotoTile(
                      color: colors[i],
                      overlay: i == 5 ? '+${post.extraPhotos}' : null,
                    ),
                  ),
                  if (i < count - 1) const SizedBox(width: 6),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _GridRows extends StatelessWidget {
  const _GridRows({required this.colors});

  final List<Color> colors;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Expanded(
          child: Row(
            children: [
              Expanded(child: FoodPhotoTile(color: colors[0])),
              const SizedBox(width: 6),
              Expanded(child: FoodPhotoTile(color: colors[1])),
            ],
          ),
        ),
        const SizedBox(height: 6),
        Expanded(
          child: Row(
            children: [
              Expanded(child: FoodPhotoTile(color: colors[2])),
              const SizedBox(width: 6),
              Expanded(child: FoodPhotoTile(color: colors[3])),
            ],
          ),
        ),
      ],
    );
  }
}

class FoodPhotoTile extends StatelessWidget {
  const FoodPhotoTile({
    super.key,
    required this.color,
    this.label,
    this.overlay,
    this.placeLabel,
  });

  final Color color;
  final String? label;
  final String? overlay;
  final String? placeLabel;

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(18),
      child: Stack(
        fit: StackFit.expand,
        children: [
          DecoratedBox(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [
                  Color.alphaBlend(Colors.white.withValues(alpha: 0.18), color),
                  color,
                  Color.alphaBlend(Colors.black.withValues(alpha: 0.12), color),
                ],
              ),
            ),
          ),
          Positioned(
            right: 12,
            bottom: 10,
            child: Icon(
              Icons.restaurant_rounded,
              color: Colors.white.withValues(alpha: 0.34),
              size: 34,
            ),
          ),
          if (placeLabel != null)
            Positioned(
              left: 10,
              top: 10,
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 6),
                decoration: BoxDecoration(
                  color: const Color(0xDD3A302A),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(
                      Icons.location_on_rounded,
                      size: 13,
                      color: AppTheme.accent,
                    ),
                    const SizedBox(width: 4),
                    Text(
                      placeLabel!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 11.5,
                        fontWeight: FontWeight.w900,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          if (label != null)
            Positioned(
              left: 12,
              right: 12,
              bottom: 10,
              child: Text(
                label!,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w900,
                ),
              ),
            ),
          if (overlay != null)
            Container(
              color: Colors.black.withValues(alpha: 0.38),
              alignment: Alignment.center,
              child: Text(
                overlay!,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 24,
                  fontWeight: FontWeight.w900,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

enum PillTone { place, price, distance, tag }

class SoftPill extends StatelessWidget {
  const SoftPill({
    super.key,
    this.icon,
    required this.label,
    this.tone = PillTone.tag,
  });

  final IconData? icon;
  final String label;
  final PillTone tone;

  @override
  Widget build(BuildContext context) {
    final colors = switch (tone) {
      PillTone.place => const _PillColors(
        background: Color(0xFFE9F6EF),
        foreground: AppTheme.brand,
        border: Color(0xFFCDE8DA),
      ),
      PillTone.price => const _PillColors(
        background: Color(0xFFFFF4DC),
        foreground: Color(0xFF9A6514),
        border: Color(0xFFFFE0A6),
      ),
      PillTone.distance => const _PillColors(
        background: Color(0xFFEAF3FF),
        foreground: Color(0xFF2F6DA8),
        border: Color(0xFFCFE2FA),
      ),
      PillTone.tag => const _PillColors(
        background: Color(0xFFF4F1EE),
        foreground: Color(0xFF6D625B),
        border: Color(0xFFE8DFD8),
      ),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
      decoration: BoxDecoration(
        color: colors.background,
        borderRadius: BorderRadius.circular(11),
        border: Border.all(color: colors.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 13, color: colors.foreground),
            const SizedBox(width: 3),
          ],
          Text(
            label,
            style: TextStyle(
              color: colors.foreground,
              fontWeight: FontWeight.w900,
              fontSize: 11.5,
            ),
          ),
        ],
      ),
    );
  }
}

class _PillColors {
  const _PillColors({
    required this.background,
    required this.foreground,
    required this.border,
  });

  final Color background;
  final Color foreground;
  final Color border;
}

class ActionChipButton extends StatelessWidget {
  const ActionChipButton({super.key, required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () {},
      borderRadius: BorderRadius.circular(14),
      child: Ink(
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 5),
        decoration: BoxDecoration(
          color: Colors.transparent,
          borderRadius: BorderRadius.circular(14),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 17, color: AppTheme.muted),
            const SizedBox(width: 5),
            Text(
              label,
              style: const TextStyle(
                color: AppTheme.muted,
                fontSize: 12,
                fontWeight: FontWeight.w800,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class MoreDotButton extends StatelessWidget {
  const MoreDotButton({super.key});

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: 'Thêm',
      child: InkWell(
        onTap: () {},
        borderRadius: BorderRadius.circular(12),
        child: const SizedBox(
          width: 30,
          height: 30,
          child: Icon(
            Icons.more_horiz_rounded,
            size: 20,
            color: AppTheme.muted,
          ),
        ),
      ),
    );
  }
}

class DiscoverScreen extends StatelessWidget {
  const DiscoverScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 96),
        children: [
          const DiscoverTopBar(),
          const SizedBox(height: 14),
          const DiscoverHeroCard(),
          const SizedBox(height: 14),
          const DiscoverCategoryRail(),
          const SizedBox(height: 16),
          const SectionTitle(title: 'Đang nổi ở Hải Phòng', action: 'Xem thêm'),
          const SizedBox(height: 10),
          const TrendingDishStrip(),
          const SizedBox(height: 16),
          const SectionTitle(title: 'Quán được nhắc nhiều', action: 'Bộ lọc'),
          const SizedBox(height: 10),
          MiniPlaceCard(post: samplePosts[0]),
          const SizedBox(height: 10),
          MiniPlaceCard(post: samplePosts[1]),
          const SizedBox(height: 10),
          MiniPlaceCard(post: samplePosts[2]),
        ],
      ),
    );
  }
}

class DiscoverTopBar extends StatelessWidget {
  const DiscoverTopBar({super.key});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Khám phá',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 3),
              Text(
                'Món ngon, quán hay, review thật quanh bạn.',
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            ],
          ),
        ),
        const AppIconButton(icon: Icons.map_outlined, tooltip: 'Bản đồ'),
      ],
    );
  }
}

class DiscoverHeroCard extends StatelessWidget {
  const DiscoverHeroCard({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(26),
        border: Border.all(color: AppTheme.line),
      ),
      child: Column(
        children: [
          Row(
            children: [
              Expanded(
                child: Container(
                  height: 42,
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  decoration: BoxDecoration(
                    color: const Color(0xFFFFFBF7),
                    borderRadius: BorderRadius.circular(17),
                    border: Border.all(color: AppTheme.line),
                  ),
                  child: Row(
                    children: [
                      const Icon(
                        Icons.search_rounded,
                        color: AppTheme.brand,
                        size: 20,
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          'Tìm món, quán, khu vực...',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Container(
                height: 42,
                padding: const EdgeInsets.symmetric(horizontal: 12),
                decoration: BoxDecoration(
                  color: AppTheme.brand,
                  borderRadius: BorderRadius.circular(17),
                ),
                child: const Row(
                  children: [
                    Icon(
                      Icons.location_on_rounded,
                      color: Colors.white,
                      size: 17,
                    ),
                    SizedBox(width: 4),
                    Text(
                      'Hải Phòng',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 12,
                        fontWeight: FontWeight.w900,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: const [
              Expanded(
                child: MiniMetric(value: '2.4K', label: 'review'),
              ),
              SizedBox(width: 8),
              Expanded(
                child: MiniMetric(value: '186', label: 'quán'),
              ),
              SizedBox(width: 8),
              Expanded(
                child: MiniMetric(value: '32', label: 'món hot'),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class MiniMetric extends StatelessWidget {
  const MiniMetric({super.key, required this.value, required this.label});

  final String value;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF3EB),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          Text(
            value,
            style: const TextStyle(
              color: AppTheme.ink,
              fontWeight: FontWeight.w900,
              fontSize: 15,
            ),
          ),
          Text(
            label,
            style: Theme.of(
              context,
            ).textTheme.bodyMedium?.copyWith(fontSize: 11),
          ),
        ],
      ),
    );
  }
}

class DiscoverCategoryRail extends StatelessWidget {
  const DiscoverCategoryRail({super.key});

  @override
  Widget build(BuildContext context) {
    final items = [
      ('Món địa phương', Icons.ramen_dining_rounded),
      ('Ăn sáng', Icons.wb_twilight_rounded),
      ('Hải sản', Icons.set_meal_rounded),
      ('Quán mới', Icons.fiber_new_rounded),
    ];

    return SizedBox(
      height: 82,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        itemCount: items.length,
        separatorBuilder: (_, _) => const SizedBox(width: 10),
        itemBuilder: (context, index) {
          return Container(
            width: 112,
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(22),
              border: Border.all(color: AppTheme.line),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(items[index].$2, color: AppTheme.brand, size: 20),
                const Spacer(),
                Text(
                  items[index].$1,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: AppTheme.ink,
                    fontSize: 12,
                    fontWeight: FontWeight.w900,
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}

class SectionTitle extends StatelessWidget {
  const SectionTitle({super.key, required this.title, required this.action});

  final String title;
  final String action;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Text(title, style: Theme.of(context).textTheme.titleMedium),
        const Spacer(),
        Text(
          action,
          style: const TextStyle(
            color: AppTheme.brand,
            fontSize: 12,
            fontWeight: FontWeight.w900,
          ),
        ),
      ],
    );
  }
}

class TrendingDishStrip extends StatelessWidget {
  const TrendingDishStrip({super.key});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 132,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        itemCount: samplePosts.length,
        separatorBuilder: (_, _) => const SizedBox(width: 10),
        itemBuilder: (context, index) =>
            TrendingDishCard(post: samplePosts[index]),
      ),
    );
  }
}

class TrendingDishCard extends StatelessWidget {
  const TrendingDishCard({super.key, required this.post});

  final FoodPost post;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 154,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(22),
        border: Border.all(color: AppTheme.line),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: FoodPhotoTile(
              color: post.photoColors.first,
              label: post.dishName,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            post.dishName,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontWeight: FontWeight.w900),
          ),
          Text(
            '${post.reviewCount} review',
            style: Theme.of(
              context,
            ).textTheme.bodyMedium?.copyWith(fontSize: 11),
          ),
        ],
      ),
    );
  }
}

class MiniPlaceCard extends StatelessWidget {
  const MiniPlaceCard({super.key, required this.post});

  final FoodPost post;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            SizedBox(
              width: 82,
              height: 82,
              child: FoodPhotoTile(color: post.photoColors.first),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    post.place,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${post.dishName} • ${post.area}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      const Icon(
                        Icons.star_rounded,
                        size: 16,
                        color: AppTheme.accent,
                      ),
                      const SizedBox(width: 3),
                      Text(
                        post.rating,
                        style: const TextStyle(fontWeight: FontWeight.w900),
                      ),
                      const SizedBox(width: 10),
                      Text(
                        post.price,
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class SavedScreen extends StatelessWidget {
  const SavedScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 18, 16, 96),
        children: [
          Text('Đã lưu', style: Theme.of(context).textTheme.headlineSmall),
          const SizedBox(height: 6),
          Text(
            'Các review và địa điểm bạn muốn quay lại.',
            style: Theme.of(context).textTheme.bodyMedium,
          ),
          const SizedBox(height: 18),
          for (final post in samplePosts.take(2)) ...[
            FoodPostCard(post: post),
            const SizedBox(height: 14),
          ],
        ],
      ),
    );
  }
}

class NotificationScreen extends StatelessWidget {
  const NotificationScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final items = [
      ('Minh Anh thích review bánh đa cua của bạn', '2 phút trước'),
      ('Có 4 review mới ở Hải Phòng', '18 phút trước'),
      ('Địa điểm bạn lưu vừa có cập nhật giờ mở cửa', 'Hôm qua'),
    ];

    return SafeArea(
      child: ListView.separated(
        padding: const EdgeInsets.fromLTRB(16, 18, 16, 96),
        itemCount: items.length + 1,
        separatorBuilder: (_, _) => const SizedBox(height: 10),
        itemBuilder: (context, index) {
          if (index == 0) {
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Tin báo',
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 8),
              ],
            );
          }
          final item = items[index - 1];
          return Card(
            child: ListTile(
              leading: CircleAvatar(
                backgroundColor: AppTheme.brandSoft,
                child: Icon(
                  index == 2
                      ? Icons.location_on_rounded
                      : Icons.notifications_rounded,
                  color: AppTheme.brand,
                ),
              ),
              title: Text(
                item.$1,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              subtitle: Text(item.$2),
              trailing: const Icon(Icons.chevron_right_rounded),
            ),
          );
        },
      ),
    );
  }
}

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 96),
        children: const [
          ProfileHeaderCard(),
          SizedBox(height: 12),
          ProfileStatsRow(),
          SizedBox(height: 14),
          ProfileTabs(),
          SizedBox(height: 12),
          ProfileActivityCard(),
          SizedBox(height: 14),
          ProfileMenuGroup(),
        ],
      ),
    );
  }
}

class ProfileHeaderCard extends StatelessWidget {
  const ProfileHeaderCard({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          children: [
            Row(
              children: [
                const CircleAvatar(
                  radius: 30,
                  backgroundColor: AppTheme.brand,
                  child: Text(
                    'A',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 24,
                      fontWeight: FontWeight.w900,
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'AnNgon User',
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      Text(
                        '@anngon • Hải Phòng',
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ],
                  ),
                ),
                const AppIconButton(
                  icon: Icons.settings_rounded,
                  tooltip: 'Cài đặt',
                ),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: () {},
                    icon: const Icon(Icons.edit_rounded, size: 16),
                    label: const Text('Sửa hồ sơ'),
                    style: OutlinedButton.styleFrom(
                      foregroundColor: AppTheme.ink,
                      side: const BorderSide(color: AppTheme.line),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(15),
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Container(
                  height: 40,
                  width: 44,
                  decoration: BoxDecoration(
                    color: const Color(0xFFFFF3EB),
                    borderRadius: BorderRadius.circular(15),
                  ),
                  child: const Icon(
                    Icons.share_outlined,
                    color: AppTheme.brand,
                    size: 19,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class ProfileStatsRow extends StatelessWidget {
  const ProfileStatsRow({super.key});

  @override
  Widget build(BuildContext context) {
    return const Row(
      children: [
        Expanded(
          child: MetricBox(value: '28', label: 'Bài viết'),
        ),
        SizedBox(width: 8),
        Expanded(
          child: MetricBox(value: '1.2K', label: 'Theo dõi'),
        ),
        SizedBox(width: 8),
        Expanded(
          child: MetricBox(value: '96', label: 'Đã lưu'),
        ),
      ],
    );
  }
}

class ProfileTabs extends StatelessWidget {
  const ProfileTabs({super.key});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: const [
        Expanded(child: ProfileTabPill(label: 'Bài viết', selected: true)),
        SizedBox(width: 8),
        Expanded(child: ProfileTabPill(label: 'Đã lưu', selected: false)),
        SizedBox(width: 8),
        Expanded(child: ProfileTabPill(label: 'Địa điểm', selected: false)),
      ],
    );
  }
}

class ProfileTabPill extends StatelessWidget {
  const ProfileTabPill({
    super.key,
    required this.label,
    required this.selected,
  });

  final String label;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 34,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: selected ? AppTheme.brand : Colors.white,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: selected ? AppTheme.brand : AppTheme.line),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: selected ? Colors.white : AppTheme.ink,
          fontSize: 12,
          fontWeight: FontWeight.w900,
        ),
      ),
    );
  }
}

class ProfileActivityCard extends StatelessWidget {
  const ProfileActivityCard({super.key});

  @override
  Widget build(BuildContext context) {
    return FoodPostCard(post: samplePosts[0]);
  }
}

class ProfileMenuGroup extends StatelessWidget {
  const ProfileMenuGroup({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Column(
        children: const [
          SimpleSetting(
            icon: Icons.devices_rounded,
            title: 'Thiết bị đăng nhập',
          ),
          Divider(height: 1, color: AppTheme.line),
          SimpleSetting(icon: Icons.block_rounded, title: 'Chặn và tắt tiếng'),
          Divider(height: 1, color: AppTheme.line),
          SimpleSetting(
            icon: Icons.privacy_tip_rounded,
            title: 'Quyền riêng tư',
          ),
        ],
      ),
    );
  }
}

class MetricBox extends StatelessWidget {
  const MetricBox({super.key, required this.value, required this.label});

  final String value;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: AppTheme.line),
      ),
      child: Column(
        children: [
          Text(value, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 2),
          Text(label, style: Theme.of(context).textTheme.bodyMedium),
        ],
      ),
    );
  }
}

class SimpleSetting extends StatelessWidget {
  const SimpleSetting({super.key, required this.icon, required this.title});

  final IconData icon;
  final String title;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: Icon(icon, color: AppTheme.brand),
        title: Text(title, style: Theme.of(context).textTheme.titleMedium),
        trailing: const Icon(Icons.chevron_right_rounded),
        onTap: () {},
      ),
    );
  }
}

class CreateReviewScreenV2 extends StatefulWidget {
  const CreateReviewScreenV2({super.key});

  @override
  State<CreateReviewScreenV2> createState() => _CreateReviewScreenV2State();
}

class _CreateReviewScreenV2State extends State<CreateReviewScreenV2> {
  final contentController = TextEditingController();
  final dishController = TextEditingController();
  final priceController = TextEditingController();
  final tagsController = TextEditingController();
  final placeController = TextEditingController();
  bool publishing = false;

  @override
  void dispose() {
    contentController.dispose();
    dishController.dispose();
    priceController.dispose();
    tagsController.dispose();
    placeController.dispose();
    super.dispose();
  }

  List<String> get hashtags {
    final raw = [dishController.text, ...tagsController.text.split(',')];
    return raw
        .map((tag) => tag.trim().replaceFirst(RegExp(r'^#'), ''))
        .where((tag) => tag.isNotEmpty)
        .take(10)
        .toList();
  }

  String get composedContent {
    final lines = <String>[
      contentController.text.trim(),
      if (dishController.text.trim().isNotEmpty)
        'Món: ${dishController.text.trim()}',
      if (priceController.text.trim().isNotEmpty)
        'Giá tham khảo: ${priceController.text.trim()}',
    ].where((line) => line.isNotEmpty).toList();
    return lines.join('\n');
  }

  Future<void> publish() async {
    if (publishing) return;
    final content = composedContent;
    if (content.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Bạn nhập vài dòng review trước nhé.')),
      );
      return;
    }
    setState(() => publishing = true);
    try {
      final placeId = await authController.api.createPlaceByName(
        placeController.text,
      );
      await authController.api.createPost(
        content: content,
        hashtags: hashtags,
        placeId: placeId,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Đã đăng review lên backend.')),
      );
      Navigator.of(context).pop();
    } catch (e) {
      if (!mounted) return;
      final message = e is ApiException ? e.message : e.toString();
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(message)));
    } finally {
      if (mounted) setState(() => publishing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.paper,
      appBar: AppBar(
        leading: IconButton(
          onPressed: () => Navigator.of(context).pop(),
          icon: const Icon(Icons.arrow_back_rounded),
          tooltip: 'Quay lại',
        ),
        title: const Text(
          'Tạo review',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
        ),
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
                children: [
                  _ReviewTextCard(controller: contentController),
                  const SizedBox(height: 12),
                  _ReviewMetaCard(
                    dishController: dishController,
                    priceController: priceController,
                    tagsController: tagsController,
                  ),
                  const SizedBox(height: 12),
                  const PhotoPickerPreview(),
                  const SizedBox(height: 12),
                  _PlaceInputCard(controller: placeController),
                ],
              ),
            ),
            _PublishBarV2(publishing: publishing, onPublish: publish),
          ],
        ),
      ),
    );
  }
}

class _ReviewTextCard extends StatelessWidget {
  const _ReviewTextCard({required this.controller});

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const CircleAvatar(
                  radius: 19,
                  backgroundColor: AppTheme.brandSoft,
                  child: Icon(Icons.person_rounded, color: AppTheme.brand),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Review mới',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      Text(
                        'Công khai trên feed',
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ],
                  ),
                ),
                const Icon(
                  Icons.public_rounded,
                  color: AppTheme.muted,
                  size: 18,
                ),
              ],
            ),
            const SizedBox(height: 12),
            TextField(
              controller: controller,
              minLines: 4,
              maxLines: 7,
              decoration: _softInputDecoration(
                context,
                'Món này ngon ở điểm nào? Có đáng quay lại không?',
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ReviewMetaCard extends StatelessWidget {
  const _ReviewMetaCard({
    required this.dishController,
    required this.priceController,
    required this.tagsController,
  });

  final TextEditingController dishController;
  final TextEditingController priceController;
  final TextEditingController tagsController;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: [
            _InlineMetaField(
              icon: Icons.restaurant_menu_rounded,
              title: 'Món đã ăn',
              hint: 'Bánh đa cua, bún cá cay...',
              controller: dishController,
            ),
            const Divider(height: 18, color: AppTheme.line),
            _InlineMetaField(
              icon: Icons.payments_rounded,
              title: 'Giá tham khảo',
              hint: '35.000đ hoặc 30-50k',
              controller: priceController,
            ),
            const Divider(height: 18, color: AppTheme.line),
            _InlineMetaField(
              icon: Icons.sell_rounded,
              title: 'Tag cảm nhận',
              hint: 'đậm vị, giá tốt, nên thử...',
              controller: tagsController,
            ),
          ],
        ),
      ),
    );
  }
}

class _InlineMetaField extends StatelessWidget {
  const _InlineMetaField({
    required this.icon,
    required this.title,
    required this.hint,
    required this.controller,
  });

  final IconData icon;
  final String title;
  final String hint;
  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 34,
          height: 34,
          decoration: BoxDecoration(
            color: AppTheme.brandSoft,
            borderRadius: BorderRadius.circular(13),
          ),
          child: Icon(icon, color: AppTheme.brand, size: 18),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(title, style: Theme.of(context).textTheme.titleMedium),
              TextField(
                controller: controller,
                minLines: 1,
                maxLines: 1,
                textInputAction: TextInputAction.next,
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: AppTheme.ink,
                  fontWeight: FontWeight.w700,
                ),
                decoration: InputDecoration(
                  isDense: true,
                  hintText: hint,
                  hintStyle: Theme.of(context).textTheme.bodyMedium,
                  border: InputBorder.none,
                  contentPadding: const EdgeInsets.only(top: 2),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _PlaceInputCard extends StatelessWidget {
  const _PlaceInputCard({required this.controller});

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Container(
              width: 42,
              height: 42,
              decoration: BoxDecoration(
                color: AppTheme.brandSoft,
                borderRadius: BorderRadius.circular(16),
              ),
              child: const Icon(
                Icons.location_on_rounded,
                color: AppTheme.brand,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Gắn quán / địa điểm',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  TextField(
                    controller: controller,
                    minLines: 1,
                    maxLines: 1,
                    decoration: InputDecoration(
                      isDense: true,
                      hintText: 'Nhập tên quán nếu muốn gắn',
                      hintStyle: Theme.of(context).textTheme.bodyMedium,
                      border: InputBorder.none,
                      contentPadding: const EdgeInsets.only(top: 3),
                    ),
                  ),
                ],
              ),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
              decoration: BoxDecoration(
                color: AppTheme.brand,
                borderRadius: BorderRadius.circular(13),
              ),
              child: const Text(
                'Gắn',
                style: TextStyle(
                  color: Colors.white,
                  fontSize: 12,
                  fontWeight: FontWeight.w900,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PublishBarV2 extends StatelessWidget {
  const _PublishBarV2({required this.publishing, required this.onPublish});

  final bool publishing;
  final VoidCallback onPublish;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 10, 16, 12),
      decoration: const BoxDecoration(
        color: AppTheme.paper,
        border: Border(top: BorderSide(color: AppTheme.line)),
      ),
      child: Row(
        children: [
          Expanded(
            child: Text(
              'Nháp tự lưu',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
          FilledButton.icon(
            onPressed: publishing ? null : onPublish,
            icon: publishing
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: Colors.white,
                    ),
                  )
                : const Icon(Icons.send_rounded, size: 18),
            label: Text(publishing ? 'Đang đăng' : 'Đăng review'),
            style: FilledButton.styleFrom(
              minimumSize: const Size(0, 42),
              padding: const EdgeInsets.symmetric(horizontal: 16),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

InputDecoration _softInputDecoration(BuildContext context, String hint) {
  return InputDecoration(
    hintText: hint,
    hintStyle: Theme.of(context).textTheme.bodyMedium,
    filled: true,
    fillColor: Colors.white,
    contentPadding: const EdgeInsets.all(14),
    border: OutlineInputBorder(
      borderRadius: BorderRadius.circular(20),
      borderSide: const BorderSide(color: AppTheme.line),
    ),
    enabledBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(20),
      borderSide: const BorderSide(color: AppTheme.line),
    ),
    focusedBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(20),
      borderSide: const BorderSide(color: AppTheme.brand),
    ),
  );
}

class CreateReviewScreen extends StatelessWidget {
  const CreateReviewScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.paper,
      appBar: AppBar(
        leading: IconButton(
          onPressed: () => Navigator.of(context).pop(),
          icon: const Icon(Icons.arrow_back_rounded),
          tooltip: 'Quay lại',
        ),
        title: const Text(
          'Tạo review',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
        ),
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
                children: const [
                  ReviewDraftCard(),
                  SizedBox(height: 12),
                  ReviewQuickFields(),
                  SizedBox(height: 12),
                  PhotoPickerPreview(),
                  SizedBox(height: 12),
                  PlaceAttachCard(),
                ],
              ),
            ),
            const PublishBar(),
          ],
        ),
      ),
    );
  }
}

class ReviewDraftCard extends StatelessWidget {
  const ReviewDraftCard({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const CircleAvatar(
                  radius: 19,
                  backgroundColor: AppTheme.brandSoft,
                  child: Icon(Icons.person_rounded, color: AppTheme.brand),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Review mới',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      Text(
                        'Hải Phòng • công khai',
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ],
                  ),
                ),
                const Icon(
                  Icons.public_rounded,
                  color: AppTheme.muted,
                  size: 18,
                ),
              ],
            ),
            const SizedBox(height: 12),
            TextField(
              minLines: 4,
              maxLines: 7,
              decoration: InputDecoration(
                hintText: 'Món này ngon ở điểm nào? Có đáng quay lại không?',
                hintStyle: Theme.of(context).textTheme.bodyMedium,
                filled: true,
                fillColor: Colors.white,
                contentPadding: const EdgeInsets.all(14),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(20),
                  borderSide: const BorderSide(color: AppTheme.line),
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(20),
                  borderSide: const BorderSide(color: AppTheme.line),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(20),
                  borderSide: const BorderSide(color: AppTheme.brand),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class ReviewQuickFields extends StatelessWidget {
  const ReviewQuickFields({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: const [
            CompactReviewRow(
              icon: Icons.restaurant_menu_rounded,
              title: 'Món đã ăn',
              value: 'Chọn món hoặc nhập tên món',
            ),
            Divider(height: 18, color: AppTheme.line),
            CompactReviewRow(
              icon: Icons.payments_rounded,
              title: 'Giá tham khảo',
              value: 'Thêm khoảng giá',
            ),
            Divider(height: 18, color: AppTheme.line),
            CompactReviewRow(
              icon: Icons.sell_rounded,
              title: 'Tag cảm nhận',
              value: 'đậm vị, giá tốt, nên thử...',
            ),
          ],
        ),
      ),
    );
  }
}

class CompactReviewRow extends StatelessWidget {
  const CompactReviewRow({
    super.key,
    required this.icon,
    required this.title,
    required this.value,
  });

  final IconData icon;
  final String title;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 34,
          height: 34,
          decoration: BoxDecoration(
            color: AppTheme.brandSoft,
            borderRadius: BorderRadius.circular(13),
          ),
          child: Icon(icon, color: AppTheme.brand, size: 18),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(title, style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 2),
              Text(
                value,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            ],
          ),
        ),
        const Icon(Icons.chevron_right_rounded, color: AppTheme.muted),
      ],
    );
  }
}

class PhotoPickerPreview extends StatelessWidget {
  const PhotoPickerPreview({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    'Ảnh món ăn',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                Text(
                  'Tối đa 6 ảnh',
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
              ],
            ),
            const SizedBox(height: 10),
            SizedBox(
              height: 82,
              child: ListView(
                scrollDirection: Axis.horizontal,
                children: const [
                  AddPhotoTile(),
                  SizedBox(width: 8),
                  PhotoHintTile(color: Color(0xFFFFE6D8), label: 'món chính'),
                  SizedBox(width: 8),
                  PhotoHintTile(color: Color(0xFFFFF2D5), label: 'menu/giá'),
                  SizedBox(width: 8),
                  PhotoHintTile(color: Color(0xFFEAF7EF), label: 'không gian'),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class AddPhotoTile extends StatelessWidget {
  const AddPhotoTile({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 92,
      decoration: BoxDecoration(
        color: AppTheme.brand,
        borderRadius: BorderRadius.circular(20),
      ),
      child: const Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.add_photo_alternate_rounded, color: Colors.white),
          SizedBox(height: 5),
          Text(
            'Thêm ảnh',
            style: TextStyle(
              color: Colors.white,
              fontSize: 12,
              fontWeight: FontWeight.w900,
            ),
          ),
        ],
      ),
    );
  }
}

class PhotoHintTile extends StatelessWidget {
  const PhotoHintTile({super.key, required this.color, required this.label});

  final Color color;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 92,
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: Colors.white),
      ),
      alignment: Alignment.center,
      child: Text(
        label,
        style: const TextStyle(
          color: AppTheme.muted,
          fontSize: 11.5,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}

class PlaceAttachCard extends StatelessWidget {
  const PlaceAttachCard({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Container(
              width: 42,
              height: 42,
              decoration: BoxDecoration(
                color: AppTheme.brandSoft,
                borderRadius: BorderRadius.circular(16),
              ),
              child: const Icon(
                Icons.location_on_rounded,
                color: AppTheme.brand,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Gắn quán / địa điểm',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  Text(
                    'Tìm quán hoặc đề xuất địa điểm mới',
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ],
              ),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
              decoration: BoxDecoration(
                color: AppTheme.brand,
                borderRadius: BorderRadius.circular(13),
              ),
              child: const Text(
                'Gắn',
                style: TextStyle(
                  color: Colors.white,
                  fontSize: 12,
                  fontWeight: FontWeight.w900,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class PublishBar extends StatelessWidget {
  const PublishBar({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 10, 16, 12),
      decoration: const BoxDecoration(
        color: AppTheme.paper,
        border: Border(top: BorderSide(color: AppTheme.line)),
      ),
      child: Row(
        children: [
          Expanded(
            child: Text(
              'Nháp tự lưu',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
          FilledButton.icon(
            onPressed: () {},
            icon: const Icon(Icons.send_rounded, size: 18),
            label: const Text('Đăng review'),
            style: FilledButton.styleFrom(
              minimumSize: const Size(0, 42),
              padding: const EdgeInsets.symmetric(horizontal: 16),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class AssistMetric extends StatelessWidget {
  const AssistMetric({super.key, required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xFFE9F7EF),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          Text(
            value,
            style: const TextStyle(
              color: AppTheme.trust,
              fontWeight: FontWeight.w900,
            ),
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: Theme.of(
              context,
            ).textTheme.bodyMedium?.copyWith(fontSize: 11),
          ),
        ],
      ),
    );
  }
}

class FoodPost {
  const FoodPost({
    required this.author,
    required this.city,
    required this.area,
    required this.timeAgo,
    required this.content,
    required this.place,
    required this.price,
    required this.distance,
    required this.rating,
    required this.trustScore,
    required this.reviewCount,
    required this.dishName,
    required this.likes,
    required this.comments,
    required this.avatarColor,
    required this.photoColors,
    required this.tags,
    this.extraPhotos = 0,
  });

  factory FoodPost.fromApi(Map<String, dynamic> json) {
    final userId = (json['user_id'] as num?)?.toInt();
    final placeId = (json['place_id'] as num?)?.toInt();
    final provinceId = (json['province_id'] as num?)?.toInt();
    final hashtags = (json['hashtags'] as List<dynamic>? ?? const [])
        .map((tag) => tag.toString())
        .where((tag) => tag.isNotEmpty)
        .toList();
    final images = json['images'] as List<dynamic>? ?? const [];
    final createdAt = DateTime.tryParse(json['created_at']?.toString() ?? '');
    final likeCount = (json['like_count'] as num?)?.toInt() ?? 0;
    final commentCount = (json['comment_count'] as num?)?.toInt() ?? 0;

    return FoodPost(
      author: userId == null ? 'Foodie địa phương' : 'Foodie #$userId',
      city: provinceId == null ? 'Việt Nam' : 'Tỉnh #$provinceId',
      area: placeId == null ? 'Chưa gắn địa điểm' : 'Place #$placeId',
      timeAgo: _timeAgo(createdAt),
      content: json['content']?.toString() ?? '',
      place: placeId == null ? 'Chưa gắn quán' : 'Địa điểm #$placeId',
      price: _extractPrice(json['content']?.toString() ?? ''),
      distance: 'gần bạn',
      rating: '4.${(likeCount % 8) + 1}',
      trustScore: '${88 + (likeCount % 10)}%',
      reviewCount: '${likeCount + commentCount + 1}',
      dishName: hashtags.isNotEmpty ? hashtags.first : 'Review món',
      likes: '$likeCount',
      comments: '$commentCount',
      avatarColor: _avatarColor(userId ?? 0),
      photoColors: images.isEmpty
          ? const [Color(0xFFFFE6D8)]
          : List<Color>.generate(
              images.length.clamp(1, 6),
              (index) => _photoColor(index),
            ),
      tags: hashtags.take(3).toList(),
      extraPhotos: images.length > 6 ? images.length - 6 : 0,
    );
  }

  final String author;
  final String city;
  final String area;
  final String timeAgo;
  final String content;
  final String place;
  final String price;
  final String distance;
  final String rating;
  final String trustScore;
  final String reviewCount;
  final String dishName;
  final String likes;
  final String comments;
  final Color avatarColor;
  final List<Color> photoColors;
  final List<String> tags;
  final int extraPhotos;
}

String _timeAgo(DateTime? createdAt) {
  if (createdAt == null) return 'vừa xong';
  final diff = DateTime.now().difference(createdAt.toLocal());
  if (diff.inMinutes < 1) return 'vừa xong';
  if (diff.inMinutes < 60) return '${diff.inMinutes} phút';
  if (diff.inHours < 24) return '${diff.inHours} giờ';
  return '${diff.inDays} ngày';
}

String _extractPrice(String content) {
  final match = RegExp(r'Giá tham khảo:\s*([^\n]+)').firstMatch(content);
  return match?.group(1)?.trim() ?? 'chưa rõ giá';
}

Color _avatarColor(int seed) {
  const colors = [
    Color(0xFFC4472F),
    Color(0xFF6F4A2F),
    Color(0xFF8A5B36),
    Color(0xFF2F6B4B),
  ];
  return colors[seed.abs() % colors.length];
}

Color _photoColor(int index) {
  const colors = [
    Color(0xFFFFE6D8),
    Color(0xFFFFF2D5),
    Color(0xFFEAF7EF),
    Color(0xFFE8EEF9),
    Color(0xFFF7E7EF),
    Color(0xFFEDE6DC),
  ];
  return colors[index % colors.length];
}

const samplePosts = [
  FoodPost(
    author: 'Linh Ăn Vặt',
    city: 'Hải Phòng',
    area: 'Lạch Tray',
    timeAgo: '12 phút',
    content:
        'Bánh đa cua tô vừa, nước dùng thơm, topping nhiều. Đi buổi sáng ngon hơn, quán đông nhưng lên món nhanh.',
    place: 'Bánh đa cua Cô Yến',
    price: '35.000đ',
    distance: '1.2 km',
    rating: '4.8',
    trustScore: '94%',
    reviewCount: '248',
    dishName: 'Bánh đa cua',
    likes: '248',
    comments: '36',
    avatarColor: Color(0xFFC4472F),
    photoColors: [Color(0xFFC4472F)],
    tags: ['đậm vị', 'ăn sáng'],
  ),
  FoodPost(
    author: 'Tuấn Local',
    city: 'Hải Phòng',
    area: 'Cát Bi',
    timeAgo: '38 phút',
    content:
        'Bún cá cay hợp ngày mưa, cá giòn, nước dùng cay vừa. Nên gọi thêm rau và quẩy nếu đi buổi trưa.',
    place: 'Bún cá cay Hàng Kênh',
    price: '45.000đ',
    distance: '2.4 km',
    rating: '4.6',
    trustScore: '92%',
    reviewCount: '179',
    dishName: 'Bún cá cay',
    likes: '91',
    comments: '14',
    avatarColor: Color(0xFF6F4A2F),
    photoColors: [
      Color(0xFFC4472F),
      Color(0xFFF4A33C),
      Color(0xFF7A4A2D),
      Color(0xFF476B42),
    ],
    tags: ['cay nhẹ', 'giá tốt'],
  ),
  FoodPost(
    author: 'Mai Review',
    city: 'Hải Phòng',
    area: 'Đồ Sơn',
    timeAgo: '1 giờ',
    content:
        'Set hải sản cho nhóm 3 người khá ổn. Ghẹ chắc, mực tươi, quán nhìn ra biển nên hợp đi cuối tuần.',
    place: 'Hải sản Bến Nghiêng',
    price: '320.000đ',
    distance: '18 km',
    rating: '4.7',
    trustScore: '89%',
    reviewCount: '312',
    dishName: 'Hải sản',
    likes: '516',
    comments: '82',
    avatarColor: Color(0xFF8A5B36),
    photoColors: [
      Color(0xFFF0A329),
      Color(0xFF4C8B61),
      Color(0xFFD66B35),
      Color(0xFFBA3E2E),
      Color(0xFF846044),
      Color(0xFF2F6B4B),
    ],
    tags: ['đi nhóm', 'view đẹp'],
    extraPhotos: 2,
  ),
];
