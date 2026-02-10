# Spring Security Patterns

## SecurityFilterChain Configuration

Use the component-based approach with `SecurityFilterChain` (Spring Security 6+):

```java
@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        return http
            .csrf(csrf -> csrf.csrfTokenRepository(CookieCsrfTokenRepository.withHttpOnlyFalse()))
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/api/public/**").permitAll()
                .requestMatchers("/api/admin/**").hasRole("ADMIN")
                .anyRequest().authenticated()
            )
            .sessionManagement(session -> session
                .sessionCreationPolicy(SessionCreationPolicy.STATELESS)
            )
            .oauth2ResourceServer(oauth2 -> oauth2.jwt(Customizer.withDefaults()))
            .build();
    }
}
```

## Method-Level Security

Enable and use method-level security annotations:

```java
@Configuration
@EnableMethodSecurity
public class MethodSecurityConfig {}

@Service
public class OrderService {

    @PreAuthorize("hasRole('ADMIN') or #userId == authentication.principal.id")
    public Order getOrder(Long userId, Long orderId) {
        // Only admins or the owning user can access
    }

    @PostAuthorize("returnObject.owner == authentication.name")
    public Document getDocument(Long id) {
        // Filter results after execution
    }
}
```

## CSRF Protection

- Enable CSRF for browser-based clients using `CookieCsrfTokenRepository`
- Disable CSRF only for stateless API-only services using JWT
- Always use `SameSite=Strict` or `SameSite=Lax` cookie attributes

## Authentication Patterns

```java
@Bean
public UserDetailsService userDetailsService(UserRepository repo) {
    return username -> repo.findByUsername(username)
        .map(user -> User.builder()
            .username(user.getUsername())
            .password(user.getPassword())
            .roles(user.getRoles().toArray(String[]::new))
            .build())
        .orElseThrow(() -> new UsernameNotFoundException("User not found: " + username));
}

@Bean
public PasswordEncoder passwordEncoder() {
    return new BCryptPasswordEncoder();
}
```

## Best Practices

- Never use `WebSecurityConfigurerAdapter` (deprecated since Spring Security 5.7)
- Always use constructor injection for security components
- Store passwords with BCrypt; never use plain text or MD5/SHA
- Use stateless sessions for REST APIs with JWT or OAuth2
- Apply the principle of least privilege: deny by default, allow explicitly
- Audit security configurations in tests using `@WithMockUser`
