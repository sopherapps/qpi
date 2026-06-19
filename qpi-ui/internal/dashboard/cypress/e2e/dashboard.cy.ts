describe("QPI Dashboard E2E Tests", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    // Visit the dashboard page
    cy.visit("/");
  });

  // FIXME: Add more tests for the dashboard, especially testing functionality

  it("should block access until logged in and allow regular user flow", () => {
    // 1. Verify Login Modal is displayed
    cy.contains("h2", "Sign in to QPI").should("be.visible");

    // 2. Click Regular User role tab (default)
    cy.contains("button", "Regular User").should("be.visible");

    // 3. Enter wrong credentials and verify error
    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("wrongpassword");
    cy.get('button[type="submit"]').click();
    cy.contains("Invalid credentials").should("be.visible");

    // 4. Enter correct credentials and log in
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();

    // 5. Verify successful login and sidebar navigation presence
    cy.contains("h1", "QPI Interface").should("be.visible");
    cy.contains("button", "Overview").should("be.visible");
    cy.contains("button", "QPU Registry").should("be.visible");
    cy.contains("button", "Jobs Console").should("be.visible");
    cy.contains("button", "Bookings").should("be.visible");
    cy.contains("button", "Profile Settings").should("be.visible");

    // Verify Admin Panel is NOT visible to standard user
    cy.contains("button", "Admin Panel").should("not.exist");

    // 6. Navigate to Jobs Console & Submit mock job
    cy.contains("button", "Jobs Console").click();
    cy.contains("h1", "Jobs Console").should("be.visible");
    
    // Select first online QPU (should be set by default, but let's check its presence)
    cy.get("select").should("be.visible").and("not.have.value", "");
    
    // Execute job
    cy.contains("button", "Execute Job").click();
    
    // Wait for completion (status becomes completed, results show)
    cy.contains("div", "completed", { timeout: 15000 }).should("be.visible");

    // Verify duration is displayed after completion
    cy.contains("span", "Duration").parent().find("span.font-mono").should("be.visible").and("contain", "s");

    // 7. Navigate to Bookings & Book a slot
    cy.contains("button", "Bookings").click();
    cy.contains("h1", "Bookings").should("be.visible");
    cy.contains("button", "Book Time Slot").click();

    // Fill dates: start in 10 minutes, end in 20 minutes
    const pad = (n: number) => n.toString().padStart(2, '0');
    const start = new Date(Date.now() + 10 * 60000);
    const end = new Date(Date.now() + 20 * 60000);
    
    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);
    cy.get('form').submit();
    
    // Slot should now show up in the scheduled list
    cy.contains("td", "user@example.com").should("be.visible");

    // 8. Logout
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.contains("h2", "Sign in to QPI").should("be.visible");
  });

  it("should support administrator dashboard management", () => {
    // 1. Log in as administrator
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();

    // 2. Verify admin panel tab button is visible
    cy.contains("button", "Admin Panel").should("be.visible");

    // 3. Register a new QPU on QPU Registry
    cy.contains("button", "QPU Registry").click();
    cy.contains("button", "Register QPU").click();
    cy.get('input[placeholder="rigetti-aspen-9"]').type("cypress-test-qpu");
    cy.get('select').select("qblox");
    cy.contains("button", "Register Unit").click();

    // Verify success screen is displayed
    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");
    cy.contains("cypress-test-qpu").should("be.visible");
    cy.contains("qblox").should("be.visible");
    cy.contains("Connection Command").should("be.visible");
    cy.contains("button", "Done").click();

    // Verify the QPU is now listed in the grid
    cy.contains("h3", "cypress-test-qpu").should("be.visible");

    // 4. Toggle QPU state on QPU Registry
    cy.contains("span", "Driver Enable Control").should("be.visible");
    // Toggle the QPU (which is Online / Enabled) to Offline
    cy.contains("button", "Online (Enabled)").click();
    // Verify it becomes Offline (Disabled)
    cy.contains("button", "Offline (Disabled)").should("be.visible");
    // Toggle it back online
    cy.contains("button", "Offline (Disabled)").click();
    cy.contains("button", "Online (Enabled)").should("be.visible");

    // 4. Compose and post broadcast announcement on Admin Panel
    cy.contains("button", "Admin Panel").click();
    cy.contains("button", "Broadcast Announcement").click();
    cy.get('input[placeholder="QPU Maintenance Schedule"]').type("Test Cypress Broadcast Title");
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]').type("Test Cypress Broadcast Description");
    cy.get('form button[type="submit"]').click();
    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // 5. Verify broadcast is visible in header dropdown
    cy.get("svg.lucide-bell").parent().click();
    cy.contains("Test Cypress Broadcast Title").should("be.visible");

    // 6. Sign Out
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.contains("h2", "Sign in to QPI").should("be.visible");
  });
});
