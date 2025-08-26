# Story Generation Testing Guide

This comprehensive guide will help you test your interactive story generation application thoroughly.

## 🧪 Story Generation Testing Guide

### **1. Basic Functionality Test**

**Start a new story:**
1. Open your browser to `http://localhost:9779`
2. Click on a genre button (Fantasy, Sci-Fi, or Historical Fiction)
3. Select a difficulty level
4. The application should generate an initial story scenario

**What to check:**
- ✅ Story text appears without errors
- ✅ Player status shows (health, conditions)
- ✅ Inventory is displayed (initially empty)
- ✅ Input field is available for responses
- ✅ Word counter works (0/15 words)

### **2. Story Interaction Test**

**Submit responses:**
1. Type a response in the input field (keep it under 15 words)
2. Click "Send" or press Enter
3. The application should generate a continuation

**What to check:**
- ✅ Response is accepted without errors
- ✅ AI generates a meaningful continuation
- ✅ Story history shows your input and AI response
- ✅ Player status updates appropriately
- ✅ Any new items appear in inventory
- ✅ Background color changes based on story mood

### **3. Input Validation Test**

**Test word limits:**
- Try submitting responses with 16+ words → Should show error
- Try submitting empty responses → Should show error
- Try submitting exactly 15 words → Should work

**Test special characters:**
- Try responses with excessive special characters → Should be handled gracefully

### **4. Genre-Specific Testing**

**Fantasy Genre:**
- Should include magic, mythical creatures, medieval elements
- Items might have properties like "magical", "cursed"
- Story should feel epic/fantastical

**Sci-Fi Genre:**
- Should include technology, space, futuristic elements
- Items might have properties like "electronic", "energy_source"
- Story should feel technological

**Historical Fiction Genre:**
- Should be set in a specific historical event
- Should include period-appropriate elements
- Should have educational/historical context

### **5. Difficulty Testing**

**Exploratory Mode:**
- Should be forgiving with minimal consequences
- Player should rarely lose health
- Story should focus on discovery

**Challenging Mode:**
- Should have clear risk/reward trade-offs
- Player health can decrease but rarely to zero
- Puzzles should be solvable

**Punishing Mode:**
- Should have severe consequences for mistakes
- Story should feel high-stakes
- Player can die (game over)

### **6. Advanced Feature Testing**

**Inventory Management:**
- Items should appear when acquired
- Items should disappear when lost
- Hovering over items should show descriptions

**Story Continuity:**
- Story should maintain consistency across turns
- Character relationships should persist
- World state should evolve logically

**PDF Download:**
- Click the download button when story ends
- Should generate a properly formatted PDF
- PDF should include story history, glossary, and metadata

### **7. Error Handling Testing**

**AI Response Issues:**
- Try triggering AI failures (if possible)
- Check that error messages are user-friendly
- Verify the application recovers gracefully

**Network Issues:**
- Test with slow/unstable connections
- Verify timeout handling

### **8. Performance Testing**

**Response Times:**
- Initial story generation: Should take ~10-20 seconds
- Subsequent responses: Should be faster (~5-10 seconds)
- Check that loading indicators work properly

**Memory Usage:**
- Visit `/health` endpoint to check memory stats
- Monitor for memory leaks during extended use

### **9. Security Testing**

**Input Validation:**
- Try various injection attempts
- Test with extremely long inputs
- Verify HTML sanitization works

**Rate Limiting:**
- Make multiple rapid requests
- Should be limited to 10 requests per minute
- Check that appropriate error messages appear

### **10. Manual Test Scenarios**

Try these specific scenarios:

1. **Combat Scenario**: "I attack the goblin with my sword"
2. **Exploration**: "I search the room carefully"
3. **Social Interaction**: "I talk to the mysterious stranger"
4. **Puzzle Solving**: "I examine the ancient runes on the wall"
5. **Item Usage**: "I drink the glowing potion"

### **11. Monitoring & Health Checks**

**Health Endpoint** (`/health`):
```bash
curl http://localhost:9779/health
```
Should return JSON with status, uptime, and memory statistics.

**Readiness Endpoint** (`/ready`):
```bash
curl http://localhost:9779/ready
```
Should return `{"status":"ready"}` when service is healthy.

### **12. Browser Developer Tools**

Use browser dev tools to check:
- Network requests are successful (status 200)
- No JavaScript errors in console
- HTMX requests complete properly
- Response times are reasonable

### **13. Common Issues to Watch For**

- **AI Response Errors**: Check logs for AI API failures
- **Template Rendering**: Ensure all dynamic content displays correctly
- **Session Management**: Verify stories maintain state across requests
- **Rate Limiting**: Don't exceed 10 requests/minute during testing

## 📊 Metrics & Monitoring

The application includes several monitoring endpoints:

- `/health` - Comprehensive health check with memory stats
- `/ready` - Readiness check for load balancers
- Rate limiting is set to 10 requests per minute per IP
- Request size is limited to 1MB

## 🔒 Security Features

- Input validation (length, content, special characters)
- Rate limiting (10 req/min per IP)
- Request size limits (1MB)
- HTML sanitization of AI responses
- AI response validation and filtering

## 🚀 Performance Expectations

- Initial story generation: 10-20 seconds
- Subsequent responses: 5-10 seconds
- Memory usage should remain stable
- No memory leaks during extended use

## 🐛 Troubleshooting

If you encounter issues:

1. Check application logs for errors
2. Verify AI API key is set correctly
3. Test with different browsers
4. Check network connectivity
5. Monitor memory usage via `/health` endpoint

## 📝 Test Results Template

Use this template to document your testing results:

```
Test Scenario: [Scenario Name]
Date: [Date]
Browser: [Browser Version]
Results:
✅ [Feature] - Working correctly
❌ [Feature] - Issue found: [Description]
⚠️ [Feature] - Warning: [Description]

Notes:
[Any additional observations]
```

Happy testing! 🎮