describe("Overview & Header — Notification Dismiss Isolation", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("still shows a notification to another user after one user dismisses it", () => {
    // 1. Admin broadcasts an announcement
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    cy.contains("button", "Broadcast Announcement").click();

    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .type("Shared Announcement");
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .type("This is a shared broadcast announcement.");
    cy.get("form").contains("button", "Broadcast Announcement").click();

    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // 2. Log out and log in as user@example.com
    cy.contains("button", "Logout").click();
    cy.contains("h2", "Welcome Back").should("be.visible");

    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // Verify the notification is visible
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Shared Announcement").should("be.visible");
      });

    // Dismiss the notification from the panel
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Shared Announcement")
          .parent()
          .parent()
          .parent()
          .trigger("mouseover")
          .within(() => {
            cy.get('button svg.lucide-x').parent().click();
          });
      });

    // Verify it's gone for this user
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Shared Announcement").should("not.exist");
      });

    // 3. Log out and log in as admin again
    cy.contains("button", "Logout").click();
    cy.contains("h2", "Welcome Back").should("be.visible");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // The notification should still be visible to admin (not dismissed by them)
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("Shared Announcement").should("be.visible");
      });
  });
});
