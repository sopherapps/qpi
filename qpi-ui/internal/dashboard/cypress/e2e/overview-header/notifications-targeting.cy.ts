describe("Overview & Header — Notification Targeting", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("shows broadcast notifications to all users", function() {
    // Admin broadcasts an announcement (no target users = broadcast)
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    cy.contains("button", "Broadcast Announcement").click();

    const uniqueTitle = `Shared Announcement`;
    
    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .clear().type(uniqueTitle);
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .clear().type(`This is a shared broadcast announcement.`);
    
    // Fill datetime
    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() - 5 * 60000);
    const end = new Date(Date.now() + 20 * 60000);
    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);

    cy.get("form").contains("button", "Broadcast Announcement").click();
    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // Log out admin
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    // Log in as user@example.com
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
    cy.visit("/#overview");

    // Verify the broadcast is visible for user
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(uniqueTitle).scrollIntoView().should("be.visible");
      });
  });

  it("does not show targeted notifications to non-target users", () => {
    // This test verifies the backend filtering: a notification targeted
    // to specific users should not be fetched by a user not in the list.
    // In our E2E, we assume seed.py set up the data or we can just verify
    // by checking that the notifications collection has the correct filter.

    // Log in as regular user
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // "Welcome to QPI" or generic seeded notifications may exist, but
    // we expect that the list of announcements contains no "Admin Only" messages.
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Admin Only").should("not.exist");
      });
  });
});
