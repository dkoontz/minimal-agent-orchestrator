# Add user logout button

## Summary
Add a logout button to the application header that clears the user session and redirects to the login page.

## Requirements
- Add a logout button to the header component
- Clicking the button should clear the session
- User should be redirected to /login after logout

## Acceptance Criteria
- [ ] Button is visible when user is logged in
- [ ] Button is hidden when user is logged out
- [ ] Clicking button clears session storage
- [ ] User is redirected to /login after logout

## Out of Scope
- Logout confirmation dialog
- Remember me functionality
- Session timeout handling

## Technical Context

### Files to Modify
- `src/components/Header.tsx` - Add logout button
- `src/hooks/useAuth.ts` - Add logout function

### Related Files
- `src/pages/Login.tsx` - Redirect target
- `src/context/AuthContext.tsx` - Session state

### Patterns to Follow
- Use existing Button component from design system
- Follow useAuth hook pattern for session management

## Testing Requirements
- Unit test for logout function in useAuth
- Integration test for redirect behavior
- Test that session storage is cleared

## Notes
- Session is stored in localStorage under 'auth_token' key
- The Header component already imports useAuth hook
